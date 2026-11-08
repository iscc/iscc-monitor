// This file adds the write/read side of the schema-agnostic iscc_index projection
// (ADR-0008): RecordProjections upserts per-leaf projection records into the
// iscc_index table, and SeqsForISCCID is the one-to-many iscc_id → []seq read.
//
// iscc_id → seq is ONE-TO-MANY (a declaration, its deletion, and any future note
// type sharing an id), so the table is keyed on the composite (hub_id, seq) — each
// hub's absolute leaf index, so two hubs both index low leaves without colliding —
// and a lookup returns a list. The raw ISCC:-prefixed iscc_id string and the raw
// inner note.$schema are stored
// verbatim and never interpreted: the index decodes nothing (no ISCC-ID codec, no
// maintype/subtype check, no known-schema list). iscc_id is stored into BOTH the
// iscc_id BLOB column (its UTF-8 bytes, so the existing iscc_index_by_iscc_id index
// drives the lookup without a codec) and the iscc_id_str TEXT column (the verbatim
// string); an empty id writes an empty BLOB + empty string, indexed verbatim too.
//
// store stays a leaf: this file uses context, database/sql, errors, and fmt (stdlib)
// only and does NOT import internal/logclient. The follower copies
// logclient.Projection → ProjectionRecord at the call site, so net/http-bearing deps
// never enter the store closure.
//
// It is an unwired-until-M2 export seam: RecordProjections has no production caller
// yet (the PollHub tile-ingestion wiring is a later slice) and SeqsForISCCID exists
// to unblock resolving a sampled iscc_id → leafIndex for VerifyInclusionEvidence.
// go vet is clean; this is not dead code.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ProjectionRecord is one entry-bundle leaf to persist into the iscc_index table —
// a store-owned plain value struct mirroring CheckpointRecord/HubKey. Its fields map
// 1:1 to the columns: HubID is the owning hub; Seq is the leaf's absolute index
// (together they form the composite (hub_id, seq) PRIMARY KEY); IsccID is the raw
// ISCC:-prefixed iscc_id string (stored into both the
// iscc_id BLOB and iscc_id_str TEXT columns); NoteSchema is the verbatim inner
// note.$schema discriminator (never validated); NoteTimestamp is the verbatim optional
// inner note.timestamp RFC-3339 string (the record's own creation/signing time,
// stored as nullable TEXT — "" → SQL NULL, never parsed, ADR-0008); RecordSHA256 is
// the record-content hash. The follower copies logclient.Projection into this struct
// at the call site.
type ProjectionRecord struct {
	HubID         int64
	Seq           uint64
	IsccID        string
	NoteSchema    string
	NoteTimestamp string
	RecordSHA256  [32]byte
}

// RecordProjections upserts a batch of projection records into iscc_index, one row
// per record keyed on the composite (hub_id, seq) PRIMARY KEY. Re-ingesting an
// already-mirrored bundle is idempotent: an existing (hub_id, seq) is overwritten in
// place via ON CONFLICT(hub_id, seq) DO UPDATE (the same row count, the latest values
// win), mirroring RecordTile's composite-PK upsert. iscc_id is bound as both UTF-8
// bytes (the BLOB) and the raw
// string (the TEXT column); an empty IsccID writes an empty BLOB + empty string, not
// NULL. note_timestamp is bound via nullStringOrNil so an absent timestamp ("") is a
// true SQL NULL distinct from a present empty string (mirroring the key-cache
// nullable-TEXT idiom). An empty slice is a no-op returning nil. Single-writer
// discipline holds (ADR-0005/0007): plain ExecContext per row on the capped pool, no
// explicit transaction (matching RecordTile).
func (s *Store) RecordProjections(ctx context.Context, recs []ProjectionRecord) error {
	for _, r := range recs {
		_, err := s.db.ExecContext(ctx,
			"INSERT INTO iscc_index (hub_id, seq, iscc_id, iscc_id_str, note_schema, note_timestamp, record_sha256) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?) "+
				"ON CONFLICT(hub_id, seq) DO UPDATE SET "+
				"iscc_id = excluded.iscc_id, "+
				"iscc_id_str = excluded.iscc_id_str, note_schema = excluded.note_schema, "+
				"note_timestamp = excluded.note_timestamp, record_sha256 = excluded.record_sha256",
			r.HubID, int64(r.Seq), []byte(r.IsccID), r.IsccID, r.NoteSchema,
			nullStringOrNil(r.NoteTimestamp), r.RecordSHA256[:],
		)
		if err != nil {
			return fmt.Errorf("store.RecordProjections: seq %d: %w", r.Seq, err)
		}
	}
	return nil
}

// RecordRow is one indexed leaf for the log-browser record list — a store-owned
// plain value the HTML record-list page renders. Seq is the leaf's absolute index;
// IsccID is the verbatim ISCC:-prefixed string (read from iscc_id_str, the empty
// string for a NULL/empty column); NoteSchema is the verbatim inner note.$schema
// discriminator; NoteTimestamp is the verbatim optional inner note.timestamp RFC-3339
// string (read from note_timestamp, "" for a NULL/empty column). All strings are
// listed verbatim and never interpreted (ADR-0008): the record list decodes nothing
// about the id, the schema, or the time.
type RecordRow struct {
	Seq           uint64
	IsccID        string
	NoteSchema    string
	NoteTimestamp string
}

// ListRecords reads a newest-first (ORDER BY seq DESC) window of a hub's indexed
// leaves for the log-browser record list, plus the hub's total indexed-record count
// so the page can render an honest "showing N of TOTAL" and decide which pagination
// links are live. It is a leaf read returning plain []RecordRow (store stays a leaf).
//
// last is the accepted-tree ceiling — the monitor's accepted tree size (FollowState
// .LastSize): only leaves with seq < last are listed, and total counts only those, so
// the list never shows a leaf the accepted checkpoint does not cover. The guard is
// exclusive (seq < last) because seq is 0-based and last is a count, matching the
// seq >= size cap every other record route applies (serveEntries / serveInclusion /
// serveVerify). On a freeze or fault ingest can leave iscc_index rows at seq >= last
// (projections are written before AdvanceAccepted), so an uncapped list would imply
// those unaccepted leaves are vouched for (coverage honesty, ADR-0001). last == 0
// (followed-but-unpolled) yields an empty list, not an error — the empty-state path.
//
// Pagination uses a seq cursor, not OFFSET, so paging stays stable under concurrent
// ingest. hasFrom distinguishes "no cursor" (start at the newest leaf below the
// ceiling) from an explicit from cursor (start at the inclusive upper-bound seq from
// and walk down) — seq 0 is a valid cursor, so from is NOT overloaded as the
// start-at-newest sentinel. n bounds the page size and must be > 0 (the handler clamps
// it before calling). iscc_id_str / note_schema / note_timestamp are read through
// sql.NullString so a NULL column degrades to "" rather than an error, and seq is
// scanned as int64 then uint64(seq) (symmetric with RecordProjections' int64(r.Seq)
// write). A hub with no indexed records returns an empty slice, total 0, and a nil
// error (an empty index is not an error — the empty-log case the record list must
// render). The id, schema, and timestamp are listed verbatim and never interpreted
// (ADR-0008).
func (s *Store) ListRecords(ctx context.Context, hubID int64, last uint64, hasFrom bool, from uint64, n int) ([]RecordRow, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM iscc_index WHERE hub_id = ? AND seq < ?", hubID, int64(last),
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store.ListRecords: count hub %d: %w", hubID, err)
	}

	query := "SELECT seq, iscc_id_str, note_schema, note_timestamp FROM iscc_index WHERE hub_id = ? AND seq < ? "
	args := []any{hubID, int64(last)}
	if hasFrom {
		query += "AND seq <= ? "
		args = append(args, int64(from))
	}
	query += "ORDER BY seq DESC LIMIT ?"
	args = append(args, n)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("store.ListRecords: hub %d: %w", hubID, err)
	}
	defer func() { _ = rows.Close() }()

	var records []RecordRow
	for rows.Next() {
		var (
			seq           int64
			isccID        sql.NullString
			noteSchema    sql.NullString
			noteTimestamp sql.NullString
		)
		if err := rows.Scan(&seq, &isccID, &noteSchema, &noteTimestamp); err != nil {
			return nil, 0, fmt.Errorf("store.ListRecords: scan: %w", err)
		}
		records = append(records, RecordRow{
			Seq:           uint64(seq),
			IsccID:        isccID.String,
			NoteSchema:    noteSchema.String,
			NoteTimestamp: noteTimestamp.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store.ListRecords: rows: %w", err)
	}
	return records, total, nil
}

// RecordAt is the single-row reader for the single-record page: it returns the one
// projection row (the verbatim iscc_id_str, note.$schema, and note.timestamp) a hub
// indexed at the absolute seq, with found reporting whether a row exists. It is a leaf
// read scoped to one (hub, seq) via QueryRowContext on the PRIMARY KEY, returning a
// plain RecordRow (store stays a leaf).
//
// An absent row is the no-row case, not an error: sql.ErrNoRows maps to (RecordRow{},
// found=false, nil err) so the caller treats "no projection indexed for this seq" as
// a plain miss (the leaf's mirrored bytes are the source of truth; the projection is
// only a derived view, ADR-0008). iscc_id_str / note_schema / note_timestamp are read
// through sql.NullString so a NULL column degrades to "" rather than an error, and seq
// is scanned as int64 then uint64(seq) (symmetric with RecordProjections' int64(r.Seq)
// write). The id, schema, and timestamp are read verbatim and never interpreted. Only
// a real query/scan fault returns a non-nil error.
func (s *Store) RecordAt(ctx context.Context, hubID int64, seq uint64) (RecordRow, bool, error) {
	var (
		rowSeq        int64
		isccID        sql.NullString
		noteSchema    sql.NullString
		noteTimestamp sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT seq, iscc_id_str, note_schema, note_timestamp FROM iscc_index WHERE hub_id = ? AND seq = ?",
		hubID, int64(seq),
	).Scan(&rowSeq, &isccID, &noteSchema, &noteTimestamp)
	if errors.Is(err, sql.ErrNoRows) {
		return RecordRow{}, false, nil
	}
	if err != nil {
		return RecordRow{}, false, fmt.Errorf("store.RecordAt: hub %d seq %d: %w", hubID, seq, err)
	}
	return RecordRow{
		Seq:           uint64(rowSeq),
		IsccID:        isccID.String,
		NoteSchema:    noteSchema.String,
		NoteTimestamp: noteTimestamp.String,
	}, true, nil
}

// RecentRecords reads up to n most-recently-indexed leaves across ALL hubs
// (realm-wide), newest first by the global seq PRIMARY KEY, bounded to each hub's
// accepted tree. It is the realm-wide read behind the dashboard's "recently
// declared" row — a single JOIN + ORDER BY query rather than a per-hub fan-out.
//
// The accepted-tree ceiling is the same seq < FollowState.LastSize cap ListRecords
// applies per hub, expressed here as a JOIN to follow_state with seq < last_size: a
// hub with no follow_state row, or a NULL/zero last_size (followed-but-unpolled, or
// projections written ahead of AdvanceAccepted on a freeze/fault), contributes
// nothing — so the row never surfaces a leaf the accepted checkpoint does not cover
// (coverage honesty, ADR-0001), and every id it returns resolves under /inclusion.
//
// It is schema-agnostic (ADR-0008): it filters NOTHING on note_schema or the id and
// interprets nothing — rows carry iscc_id_str / note_schema / note_timestamp verbatim
// for the caller (the dashboard view layer) to interpret which are declarations. n
// must be > 0 (the caller passes a fixed positive limit). iscc_id_str / note_schema /
// note_timestamp are read through sql.NullString so a NULL column degrades to ""
// rather than an error, and seq is scanned as int64 then uint64(seq) (symmetric with
// the RecordProjections write). An empty index returns a nil slice with a nil error
// (an empty index is not an error — the dashboard simply renders no row).
func (s *Store) RecentRecords(ctx context.Context, n int) ([]RecordRow, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT i.seq, i.iscc_id_str, i.note_schema, i.note_timestamp "+
			"FROM iscc_index i JOIN follow_state f ON f.hub_id = i.hub_id "+
			"WHERE i.seq < f.last_size ORDER BY i.seq DESC LIMIT ?",
		n,
	)
	if err != nil {
		return nil, fmt.Errorf("store.RecentRecords: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []RecordRow
	for rows.Next() {
		var (
			seq           int64
			isccID        sql.NullString
			noteSchema    sql.NullString
			noteTimestamp sql.NullString
		)
		if err := rows.Scan(&seq, &isccID, &noteSchema, &noteTimestamp); err != nil {
			return nil, fmt.Errorf("store.RecentRecords: scan: %w", err)
		}
		records = append(records, RecordRow{
			Seq:           uint64(seq),
			IsccID:        isccID.String,
			NoteSchema:    noteSchema.String,
			NoteTimestamp: noteTimestamp.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.RecentRecords: rows: %w", err)
	}
	return records, nil
}

// SeqsForISCCID is the one-to-many reader: it returns every seq a hub indexed under
// an iscc_id, ascending (a declaration and its later deletion share an id and come
// back as two seqs). It queries the iscc_id BLOB column with the id's UTF-8 bytes so
// the iscc_index_by_iscc_id index is used, and ORDER BY seq makes the result
// deterministic. No matches returns a nil slice with a nil error — absent is not an
// error, mirroring LookupHubKey — so the caller treats "not indexed" as a plain miss;
// only a real query/scan fault returns a non-nil error.
func (s *Store) SeqsForISCCID(ctx context.Context, hubID int64, isccID string) ([]uint64, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT seq FROM iscc_index WHERE hub_id = ? AND iscc_id = ? ORDER BY seq",
		hubID, []byte(isccID),
	)
	if err != nil {
		return nil, fmt.Errorf("store.SeqsForISCCID: hub %d: %w", hubID, err)
	}
	defer func() { _ = rows.Close() }()

	var seqs []uint64
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			return nil, fmt.Errorf("store.SeqsForISCCID: hub %d: %w", hubID, err)
		}
		seqs = append(seqs, uint64(seq))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.SeqsForISCCID: hub %d: %w", hubID, err)
	}
	return seqs, nil
}
