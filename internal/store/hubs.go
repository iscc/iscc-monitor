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
// last_size and frozen flag (from follow_state), and its coverage start
// (monitored_since_{size,time}). A hub with no follow_state row yet reports
// LastSize 0 and Frozen false (the LEFT JOIN yields NULL → zero value). Coverage
// reports Set false with zero Size / Since until the hub yields a verified
// observation, so the renderer never implies pre-coverage guarantees (ADR-0001).
type HubSummary struct {
	HubID    int64
	Domain   string
	Origin   string
	Active   bool
	LastSize uint64
	Frozen   bool
	Coverage CoverageInfo
}

// ListHubs reads one HubSummary per followed hub, ordered by hub_id, in a single
// query LEFT JOINing hubs with follow_state so a hub that has never been polled
// still appears (its last_size / frozen read as NULL/0 → zero value). It returns
// plain Go types and consults no other package, keeping store a leaf. The coverage
// start is read from hubs.monitored_since_{size,time}: a NULL size means coverage
// has not started (Set false), matching Coverage's "absent is not an error"
// convention.
func (s *Store) ListHubs(ctx context.Context) ([]HubSummary, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT h.hub_id, h.domain, h.origin, h.active, "+
			"f.last_size, f.frozen, h.monitored_since_size, h.monitored_since_time "+
			"FROM hubs h LEFT JOIN follow_state f ON f.hub_id = h.hub_id "+
			"ORDER BY h.hub_id",
	)
	if err != nil {
		return nil, fmt.Errorf("store.ListHubs: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var hubs []HubSummary
	for rows.Next() {
		var (
			h         HubSummary
			lastSize  sql.NullInt64
			frozen    sql.NullBool
			sinceSize sql.NullInt64
			sinceTime sql.NullInt64
		)
		if err := rows.Scan(
			&h.HubID, &h.Domain, &h.Origin, &h.Active,
			&lastSize, &frozen, &sinceSize, &sinceTime,
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
		hubs = append(hubs, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListHubs: rows: %w", err)
	}
	return hubs, nil
}
