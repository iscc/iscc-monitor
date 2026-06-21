// This file adds the write/read side of the schema-agnostic iscc_index projection
// (ADR-0008): RecordProjections upserts per-leaf projection records into the
// iscc_index table, and SeqsForISCCID is the one-to-many iscc_id → []seq read.
//
// iscc_id → seq is ONE-TO-MANY (a declaration, its deletion, and any future note
// type sharing an id), so seq is the PRIMARY KEY and a lookup returns a list. The
// raw ISCC:-prefixed iscc_id string and the raw inner note.$schema are stored
// verbatim and never interpreted: the index decodes nothing (no ISCC-ID codec, no
// maintype/subtype check, no known-schema list). iscc_id is stored into BOTH the
// iscc_id BLOB column (its UTF-8 bytes, so the existing iscc_index_by_iscc_id index
// drives the lookup without a codec) and the iscc_id_str TEXT column (the verbatim
// string); an empty id writes an empty BLOB + empty string, indexed verbatim too.
//
// store stays a leaf: this file uses context + fmt (stdlib) only and does NOT import
// internal/logclient. The follower copies logclient.Projection → ProjectionRecord at
// the call site, so net/http-bearing deps never enter the store closure.
//
// It is an unwired-until-M2 export seam: RecordProjections has no production caller
// yet (the PollHub tile-ingestion wiring is a later slice) and SeqsForISCCID exists
// to unblock resolving a sampled iscc_id → leafIndex for VerifyInclusionEvidence.
// go vet is clean; this is not dead code.
package store

import (
	"context"
	"database/sql"
	"fmt"
)

// ProjectionRecord is one entry-bundle leaf to persist into the iscc_index table —
// a store-owned plain value struct mirroring CheckpointRecord/HubKey. Its fields map
// 1:1 to the columns: HubID is the owning hub; Seq is the leaf's absolute index (the
// PRIMARY KEY); IsccID is the raw ISCC:-prefixed iscc_id string (stored into both the
// iscc_id BLOB and iscc_id_str TEXT columns); NoteSchema is the verbatim inner
// note.$schema discriminator (never validated); RecordSHA256 is the record-content
// hash. The follower copies logclient.Projection into this struct at the call site.
type ProjectionRecord struct {
	HubID        int64
	Seq          uint64
	IsccID       string
	NoteSchema   string
	RecordSHA256 [32]byte
}

// RecordProjections upserts a batch of projection records into iscc_index, one row
// per record keyed on seq (the PRIMARY KEY). Re-ingesting an already-mirrored bundle
// is idempotent: an existing seq is overwritten in place via ON CONFLICT(seq) DO
// UPDATE (the same row count, the latest values win), mirroring RecordTile's
// composite-PK upsert. iscc_id is bound as both UTF-8 bytes (the BLOB) and the raw
// string (the TEXT column); an empty IsccID writes an empty BLOB + empty string, not
// NULL. An empty slice is a no-op returning nil. Single-writer discipline holds
// (ADR-0005/0007): plain ExecContext per row on the capped pool, no explicit
// transaction (matching RecordTile).
func (s *Store) RecordProjections(ctx context.Context, recs []ProjectionRecord) error {
	for _, r := range recs {
		_, err := s.db.ExecContext(ctx,
			"INSERT INTO iscc_index (hub_id, seq, iscc_id, iscc_id_str, note_schema, record_sha256) "+
				"VALUES (?, ?, ?, ?, ?, ?) "+
				"ON CONFLICT(seq) DO UPDATE SET "+
				"hub_id = excluded.hub_id, iscc_id = excluded.iscc_id, "+
				"iscc_id_str = excluded.iscc_id_str, note_schema = excluded.note_schema, "+
				"record_sha256 = excluded.record_sha256",
			r.HubID, int64(r.Seq), []byte(r.IsccID), r.IsccID, r.NoteSchema, r.RecordSHA256[:],
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
// discriminator. Both strings are listed verbatim and never interpreted (ADR-0008):
// the record list decodes nothing about the id or the schema.
type RecordRow struct {
	Seq        uint64
	IsccID     string
	NoteSchema string
}

// ListRecords reads a newest-first (ORDER BY seq DESC) window of a hub's indexed
// leaves for the log-browser record list, plus the hub's total indexed-record count
// so the page can render an honest "showing N of TOTAL" and decide which pagination
// links are live. It is a leaf read returning plain []RecordRow (store stays a leaf).
//
// Pagination uses a seq cursor, not OFFSET, so paging stays stable under concurrent
// ingest: when from > 0 the page starts at the (inclusive) upper-bound seq from and
// walks down; when from == 0 it starts at the newest leaf. n bounds the page size and
// must be > 0 (the handler clamps it before calling). iscc_id_str / note_schema are
// read through sql.NullString so a NULL column degrades to "" rather than an error,
// and seq is scanned as int64 then uint64(seq) (symmetric with RecordProjections'
// int64(r.Seq) write). A hub with no indexed records returns an empty slice, total 0,
// and a nil error (an empty index is not an error — the empty-log case the record list
// must render). The id and schema are listed verbatim and never interpreted (ADR-0008).
func (s *Store) ListRecords(ctx context.Context, hubID int64, from uint64, n int) ([]RecordRow, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM iscc_index WHERE hub_id = ?", hubID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store.ListRecords: count hub %d: %w", hubID, err)
	}

	query := "SELECT seq, iscc_id_str, note_schema FROM iscc_index WHERE hub_id = ? "
	args := []any{hubID}
	if from > 0 {
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
			seq        int64
			isccID     sql.NullString
			noteSchema sql.NullString
		)
		if err := rows.Scan(&seq, &isccID, &noteSchema); err != nil {
			return nil, 0, fmt.Errorf("store.ListRecords: scan: %w", err)
		}
		records = append(records, RecordRow{
			Seq:        uint64(seq),
			IsccID:     isccID.String,
			NoteSchema: noteSchema.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("store.ListRecords: rows: %w", err)
	}
	return records, total, nil
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
