package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"fmt"
	)

// Storage provides back-end storage for BookBrowser
type Storage struct {
	// database interface
	db *sql.DB

	// sometimes a transaction needs to be shared between the XxxStorage instances, for example when saving an
	// Author as part of a Book save; the active transaction is stored here
	activeTx *sql.Tx

	// data mappers for BookBrowser's models
	Books      *BookStorage
	Authors    *AuthorStorage
	Series     *SeriesStorage
	Publishers *PublisherStorage
}

// New creates a new Storage instance from a SQLite database located at pathname.
func New(pathname string) (s *Storage, err error) {
	s = &Storage{
	}

	if s.db, err = sql.Open("sqlite3", pathname+"?cache=shared"); err != nil { // _journal_mode=WAL&
		return nil, err
	}

	defer func() {
		if err != nil {
			s.db.Close()
			s.db = nil
			s = nil
		}
	}()

	if err = s.setupSchema(); err != nil {
		return
	}

	if s.Books, err = NewBookStorage(s); err != nil {
		return
	}
	if s.Authors, err = NewAuthorStorage(s); err != nil {
		return
	}
	if s.Publishers, err = NewPublisherStorage(s); err != nil {
		return
	}

	if s.Series, err = NewSeriesStorage(s); err != nil {
		return
	}

	return
}

// setupSchema creates any tables that may be missing from the SQLite database.
func (s *Storage) setupSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS books (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	pathname VARCHAR(255) NOT NULL,
	filesize INTEGER NOT NULL,
	filemtime INTEGER NOT NULL,
	hash VARCHAR(40) NOT NULL,
	hascover INTEGER NOT NULL DEFAULT 0,
	title VARCHAR(255) NOT NULL,
	description TEXT NOT NULL,
	isbn VARCHAR(16) NOT NULL,
	publishdate INTEGER NOT NULL,
	authorid INTEGER NOT NULL,
	publisherid INTEGER,
	seriesid INTEGER,
	seriesindex INTEGER
)`,
		`CREATE TABLE IF NOT EXISTS authors (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	name VARCHAR(255) NOT NULL,
	sortname VARCHAR(255) NOT NULL,
	UNIQUE(name)
)`,
		`CREATE TABLE IF NOT EXISTS publishers (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	name VARCHAR(255) NOT NULL,
	UNIQUE(name)
)`,
		`CREATE TABLE IF NOT EXISTS series (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	name VARCHAR(255) NOT NULL,
	UNIQUE(name)
)`,
	}

	for _, query := range queries {
		_, err := s.db.Exec(query)
		if err != nil {
			err := fmt.Errorf("schema setup (%s): %v", query, err.Error())
			return err
		}
	}
	return nil
}

// Close closes the underlying database handle.
func (s *Storage) Close() error {
	return s.db.Close()
}

// Sets the "active" transaction; if set, it will be used by all data mappers instead of starting a new transaction
func (s *Storage) SetActiveTx(tx *sql.Tx) {
	s.activeTx = tx
}

// Clears the "active" transaction
func (s *Storage) ClearActiveTx() {
	s.activeTx = nil
}

// Retrieves the "active" transaction
func (s *Storage) GetActiveTx() *sql.Tx {
	return s.activeTx
}

// Retrieves the "active" transaction, or begins a new one. Returns callbacks to commit or rollback the transaction.
func (s *Storage) GetOrBeginTx() (tx *sql.Tx, commit func() error, rollback func() error, err error) {
	if s.activeTx != nil {
		tx = s.activeTx
		// if a transaction is active, stub out commit/rollback as we don't want to do either until Commit/Rollback
		// is called by the code that originally began the transaction
		commit = func() error { return nil }
		rollback = func() error { return nil }

	} else {
		tx, err = s.db.Begin()
		commit = tx.Commit
		rollback = tx.Rollback
	}
	return
}