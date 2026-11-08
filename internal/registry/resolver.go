// This file holds the realm-resolution seam the monitor's HTTP surfaces consult:
// the HubResolver behavior the certificate depends on, the AtomicHubList holder
// the binary hot-swaps on its hourly realm refresh, and the follow-entry
// derivation that turns a parsed Hub-List into the domains the follower polls. The
// parse itself stays a pure leaf in registry.go; this file adds only resolution
// and the concurrency-safe holder around it.
package registry

import (
	"fmt"
	"sync/atomic"
)

// HubResolver maps a decoded ISCC-IDv1 hub_id slot (0-4095) to the issuing hub's
// domain. Both *HubList (a fixed snapshot) and *AtomicHubList (the hot-swappable
// holder the hourly realm refresh updates) satisfy it, so the certificate handler
// depends on the resolution behavior rather than a concrete snapshot — a refresh
// is a single atomic store no in-flight request ever observes mid-resolve.
type HubResolver interface {
	Resolve(hubID uint16) (domain string, ok bool)
}

// Entries derives the follower's realm entries (bare domain + https base URL) from
// the Hub-List in document order. The 12-bit hub_id slot is intentionally dropped:
// the follower and the store key a hub by its DOMAIN (store.UpsertHub assigns its
// own surrogate id), so following needs only the domains — the slot matters solely
// to the certificate's hub_id resolution via Resolve. A nil receiver yields no
// entries. It fails closed, returning a wrapped error naming the offending url, if
// any hub's url is not a bare-host base url; a *HubList from ParseHubList never
// carries such a url (ParseHubList validates every url through the same hubDomain),
// so in practice this is total.
func (hl *HubList) Entries() ([]Entry, error) {
	if hl == nil {
		return nil, nil
	}
	entries := make([]Entry, 0, len(hl.Hubs))
	for _, h := range hl.Hubs {
		domain, err := hubDomain(h.URL)
		if err != nil {
			return nil, fmt.Errorf("registry: hub url %q: %w", h.URL, err)
		}
		entries = append(entries, Entry{Domain: domain, BaseURL: h.URL})
	}
	return entries, nil
}

// AtomicHubList is a concurrency-safe holder for the current realm Hub-List. The
// binary seeds it at startup and the hourly realm refresh swaps in a freshly
// fetched Hub-List with a single atomic store, so every in-flight certificate
// request resolves against one consistent snapshot and a refresh never tears a
// read. The zero value is not usable; construct it with NewAtomicHubList.
type AtomicHubList struct {
	p atomic.Pointer[HubList]
}

// NewAtomicHubList builds a holder seeded with hl, which may be nil — a nil
// snapshot resolves nothing, the fail-closed empty-realm state.
func NewAtomicHubList(hl *HubList) *AtomicHubList {
	a := &AtomicHubList{}
	a.p.Store(hl)
	return a
}

// Store atomically replaces the current snapshot. The hourly realm refresh calls
// it only after a successful fetch+parse, so a transient fetch failure leaves the
// last good snapshot in place — Store is never called with the failed result.
func (a *AtomicHubList) Store(hl *HubList) { a.p.Store(hl) }

// Load returns the current snapshot, or nil when none has been seeded.
func (a *AtomicHubList) Load() *HubList { return a.p.Load() }

// Resolve maps a hub_id slot through the current snapshot, failing closed to
// ("", false) when no snapshot is seeded (HubList.Resolve is nil-receiver-safe).
func (a *AtomicHubList) Resolve(hubID uint16) (domain string, ok bool) {
	return a.p.Load().Resolve(hubID)
}
