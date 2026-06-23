// This file adds the read-only hub-summary projection the dashboard renders: one
// row per followed hub joining the hubs registry row with its follow_state cursor
// and freeze flag, plus the coverage start (ADR-0001). It is a pure read — no
// writes — and returns plain Go types so store stays a leaf (database/sql + stdlib
// only; no logclient / net/http in its closure). The dashboard imports store to
// call this; store never imports the dashboard.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// HubSummary is one followed hub's store-provable state for the dashboard: its
// domain and log origin, whether it is active in the realm registry, its accepted
// last_size and frozen flag (from follow_state), its coverage start
// (monitored_since_{size,time}), and its latest-stamped-root anchor status
// (Anchor). A hub with no follow_state row yet reports LastSize 0 and Frozen false
// (the LEFT JOIN yields NULL → zero value). Coverage reports Set false with zero
// Size / Since until the hub yields a verified observation, so the renderer never
// implies pre-coverage guarantees (ADR-0001). Anchor is the OTSStatus* string of
// the hub's most-recently-stamped root, or "" when no root has been stamped — the
// honest "not anchored yet" state, never a guarantee.
//
// CheckpointObserved is the observed_at instant of the hub's newest checkpoint
// (the §3 "latest checkpoint" time on the dossier), zero when no checkpoint carries
// a recorded time. AnchorHeight is the Bitcoin block height of the hub's confirmed
// anchor (the §4 height), zero when no anchor is confirmed or the confirmed row has
// no recorded height — the dossier renders the height only when it is non-zero, so
// a zero never reads as block 0.
type HubSummary struct {
	HubID              int64
	Domain             string
	Origin             string
	Active             bool
	LastSize           uint64
	Frozen             bool
	Coverage           CoverageInfo
	Anchor             string
	CheckpointObserved time.Time
	AnchorHeight       uint64
}

// ListHubs reads one HubSummary per followed hub, ordered by hub_id, in a single
// query LEFT JOINing hubs with follow_state so a hub that has never been polled
// still appears (its last_size / frozen read as NULL/0 → zero value). It returns
// plain Go types and consults no other package, keeping store a leaf. The coverage
// start is read from hubs.monitored_since_{size,time}: a NULL size means coverage
// has not started (Set false), matching Coverage's "absent is not an error"
// convention. The Anchor status is a correlated subselect projecting the status of
// the hub's most-recently-stamped root (newest stamped_at first); a never-stamped
// hub yields SQL NULL → empty Anchor (the honest "not anchored yet" state).
//
// Two further NULL-safe correlated subselects mirror that Anchor pattern for the
// hub dossier: the newest checkpoint's observed_at (the §3 "latest checkpoint" time,
// ordered tree_size DESC, id DESC) and the btc_height of the hub's confirmed anchor
// (the §4 height, scoped to status = OTSStatusConfirmed). A hub with no recorded
// checkpoint time or no confirmed anchor yields SQL NULL → the zero value, so the
// renderer shows the honest "unknown" / "not confirmed" state rather than a
// fabricated instant or block 0.
func (s *Store) ListHubs(ctx context.Context) ([]HubSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT h.hub_id, h.domain, h.origin, h.active, "+
			"f.last_size, f.frozen, h.monitored_since_size, h.monitored_since_time, "+
			"(SELECT o.status FROM ots o WHERE o.hub_id = h.hub_id "+
			"ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1), "+
			"(SELECT c.observed_at FROM checkpoints c WHERE c.hub_id = h.hub_id "+
			"ORDER BY c.tree_size DESC, c.id DESC LIMIT 1), "+
			"(SELECT o.btc_height FROM ots o WHERE o.hub_id = h.hub_id "+
			"AND o.status = ? ORDER BY o.stamped_at DESC, o.id DESC LIMIT 1) "+
			"FROM hubs h LEFT JOIN follow_state f ON f.hub_id = h.hub_id "+
			"ORDER BY h.hub_id",
		OTSStatusConfirmed,
	)
	if err != nil {
		return nil, fmt.Errorf("store.ListHubs: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var hubs []HubSummary
	for rows.Next() {
		var (
			h            HubSummary
			lastSize     sql.NullInt64
			frozen       sql.NullBool
			sinceSize    sql.NullInt64
			sinceTime    sql.NullInt64
			anchor       sql.NullString
			observedAt   sql.NullInt64
			anchorHeight sql.NullInt64
		)
		if err := rows.Scan(
			&h.HubID, &h.Domain, &h.Origin, &h.Active,
			&lastSize, &frozen, &sinceSize, &sinceTime, &anchor, &observedAt, &anchorHeight,
		); err != nil {
			return nil, fmt.Errorf("store.ListHubs: scan: %w", err)
		}
		if lastSize.Valid {
			h.LastSize = uint64(lastSize.Int64)
		}
		h.Frozen = frozen.Bool // NULL (no follow_state row) → false
		if sinceSize.Valid {
			h.Coverage.Set = true
			h.Coverage.Size = uint64(sinceSize.Int64)
			if sinceTime.Valid {
				h.Coverage.Since = time.Unix(sinceTime.Int64, 0)
			}
		}
		h.Anchor = anchor.String // NULL (no stamped root) → ""
		if observedAt.Valid {
			h.CheckpointObserved = time.Unix(observedAt.Int64, 0)
		}
		if anchorHeight.Valid {
			h.AnchorHeight = uint64(anchorHeight.Int64)
		}
		hubs = append(hubs, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListHubs: rows: %w", err)
	}
	return hubs, nil
}
