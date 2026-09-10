// Package store keeps every model archdoc has produced for a repository, so a later run can
// answer *what changed* rather than only *what is*.
//
// The engine is SQLite through modernc.org/sqlite — a pure-Go implementation with no cgo. That
// is not a preference: cgo would cost the single static binary and cross-compilation, which are
// the reasons the tool can be handed to someone as one file.
//
// **What is durable, and what is a cache.** The database is a local index, not the archive. The
// durable record is `model.json`, committed alongside the documentation, at each commit — so the
// model as of any revision is already stored by the thing designed for storing revisions. Losing
// `.archdoc/history.db` costs speed and nothing else; it can be rebuilt by regenerating. That is
// why it is not something to commit: a binary file in git conflicts on every parallel run and
// diffs as noise.
package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/cruzambrociogl/archdoc/internal/archdoc"
)

// File is where history lives, relative to the repository being documented.
const File = ".archdoc/history.db"

// Version is one model, as it stood at one run.
type Version struct {
	ID          int64
	CreatedAt   time.Time
	Commit      string // the scanned repository's revision, when it had one (MEM-01)
	Source      string // the configuration file the model came from
	Tool        string
	Fingerprint string // content hash of the model, which is what makes a no-op run detectable
	Model       archdoc.Model
}

// Store is history for one repository.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS versions (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	created_at  TEXT    NOT NULL,
	commit_sha  TEXT    NOT NULL DEFAULT '',
	source      TEXT    NOT NULL DEFAULT '',
	tool        TEXT    NOT NULL DEFAULT '',
	fingerprint TEXT    NOT NULL,
	model       TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS versions_created_at ON versions (created_at);
`

// Open prepares history for a repository, creating the database if it is not there yet.
func Open(root string) (*Store, error) {
	path := filepath.Join(root, File)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return open(path)
}

// Memory opens a database that never touches disk. Tests use it; so could a caller that wants a
// model checked without leaving anything behind.
func Memory() (*Store, error) { return open(":memory:") }

func open(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Single writer, and the whole point of an embedded engine is that there is no second
	// process to coordinate with.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Save records a model, unless the latest version already holds exactly this one.
//
// A run that changes nothing does not create a version. That is what keeps history meaningful:
// with AC-7 guaranteeing byte-identical output across runs, recording every run would fill the
// log with entries that differ in nothing but their timestamp, and a diff between two of them
// would be empty. `changed` reports which happened.
func (s *Store) Save(m archdoc.Model, commit, tool string) (id int64, changed bool, err error) {
	encoded, err := json.Marshal(m)
	if err != nil {
		return 0, false, err
	}
	sum := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(sum[:])

	latest, err := s.Latest()
	if err != nil {
		return 0, false, err
	}
	if latest != nil && latest.Fingerprint == fingerprint {
		return latest.ID, false, nil
	}

	res, err := s.db.Exec(
		`INSERT INTO versions (created_at, commit_sha, source, tool, fingerprint, model)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		time.Now().UTC().Format(time.RFC3339), commit, m.Source, tool, fingerprint, string(encoded),
	)
	if err != nil {
		return 0, false, err
	}

	id, err = res.LastInsertId()
	return id, true, err
}

// Latest returns the most recent version, or nil when there is no history yet.
func (s *Store) Latest() (*Version, error) {
	return s.one(`SELECT id, created_at, commit_sha, source, tool, fingerprint, model
	              FROM versions ORDER BY id DESC LIMIT 1`)
}

// Get returns one version by id (MEM-03).
func (s *Store) Get(id int64) (*Version, error) {
	return s.one(`SELECT id, created_at, commit_sha, source, tool, fingerprint, model
	              FROM versions WHERE id = ?`, id)
}

// Versions lists history, newest first. The model is not decoded: a listing wants dates and
// commits, and decoding every model to print a table would be work nobody asked for.
func (s *Store) Versions(limit int) ([]Version, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(
		`SELECT id, created_at, commit_sha, source, tool, fingerprint
		 FROM versions ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Version
	for rows.Next() {
		var (
			v         Version
			createdAt string
		)
		if err := rows.Scan(&v.ID, &createdAt, &v.Commit, &v.Source, &v.Tool, &v.Fingerprint); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) one(query string, args ...any) (*Version, error) {
	var (
		v         Version
		createdAt string
		encoded   string
	)

	err := s.db.QueryRow(query, args...).Scan(
		&v.ID, &createdAt, &v.Commit, &v.Source, &v.Tool, &v.Fingerprint, &encoded)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if err := json.Unmarshal([]byte(encoded), &v.Model); err != nil {
		return nil, fmt.Errorf("version %d: %w", v.ID, err)
	}
	return &v, nil
}
