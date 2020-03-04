package storage

import (
	"github.com/sblinch/BookBrowser/booklist"
	"database/sql"
	"fmt"
)

type PublisherStorage struct {
	storage *Storage

	preparedInsert *sql.Stmt
	preparedUpdate *sql.Stmt
	preparedSelectID *sql.Stmt

	baseSelectQuery string
	baseCountQuery string
}

// publisherFields puts all of the code that maps database columns to struct fields in one place for ease of management when
// adding or removing fields.
//
// Each of these declarations each must remain in precisely the same order as the others (with the exception of the ID
// field which has special requirements noted below), and all declarations must be updated when fields are added or removed.
var publisherFields = struct {
	table string
	columns []string

	insert func(stmt *sql.Stmt, publisher *booklist.Publisher) (sql.Result, error)
	update func(stmt *sql.Stmt, publisher *booklist.Publisher) (sql.Result, error)
	scan   func(rows *sql.Rows, publisher *booklist.Publisher) error
}{
	table: "publishers",

	// id FIRST in columns
	columns: []string{"id", "name"},
	scan: func(rows *sql.Rows, publisher *booklist.Publisher) error {
		// id FIRST in scan
		return rows.Scan(&publisher.ID, &publisher.Name)
	},
	insert: func(stmt *sql.Stmt, publisher *booklist.Publisher) (sql.Result, error) {
		// id OMITTED in insert
		return stmt.Exec(publisher.Name)
	},
	update: func(stmt *sql.Stmt, publisher *booklist.Publisher) (sql.Result, error) {
		// id LAST in update
		return stmt.Exec(publisher.Name, publisher.ID)
	},
}

func NewPublisherStorage(s *Storage) (*PublisherStorage, error) {
	a := &PublisherStorage{
		storage: s,
	}

	var err error
	if a.preparedInsert, err = s.db.Prepare(buildInsertQuery(publisherFields.table,publisherFields.columns, true)); err != nil {
		return nil, err
	}
	if a.preparedUpdate, err = s.db.Prepare(buildUpdateQuery(publisherFields.table,publisherFields.columns)); err != nil {
		return nil, err
	}

	q, _ := buildSelectQuery(publisherFields.table,[]string{"id"})
	if a.preparedSelectID, err = s.db.Prepare(q+ " WHERE name=?"); err != nil {
		return nil, err
	}

	a.baseSelectQuery, a.baseCountQuery = buildSelectQuery(publisherFields.table,publisherFields.columns)

	return a, nil
}

func (a *PublisherStorage) Save(publishers ... *booklist.Publisher) error {
	tx, commit, rollback, err := a.storage.GetOrBeginTx()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			rollback()
		}
	}()

	insertStmt := tx.Stmt(a.preparedInsert)
	updateStmt := tx.Stmt(a.preparedUpdate)
	selectIDStmt := tx.Stmt(a.preparedSelectID)

	var (
		res sql.Result
		insertID int64
		existingID int
	)
	for _, publisher := range publishers {
		if publisher.ID > 0 {
			if _, err = publisherFields.update(updateStmt,publisher); err != nil {
				return fmt.Errorf("publishers, update: %v",err)
			}

		} else {
			if err := selectIDStmt.QueryRow(publisher.Name).Scan(&existingID); err == nil && existingID > 0 {
				publisher.ID = existingID
				continue
			}

			res, err = publisherFields.insert(insertStmt,publisher)
			if err == nil {
				insertID, err = res.LastInsertId()
				if err == nil {
					publisher.ID = int(insertID)
				}
			}
			if err != nil {
				return fmt.Errorf("publishers, insert: %v",err)
			}
		}
	}
	if err = commit(); err != nil {
		return fmt.Errorf("publishers, commit: %v",err)
	}
	tx = nil
	return nil
}


func (a *PublisherStorage) Count(q *Query) (int, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseCountQuery,publisherFields.columns)
	if err != nil {
		return -1, err
	}

	total := 0
	row := a.storage.db.QueryRow(query,bindValues...)
	if err := row.Scan(&total); err != nil {
		return -1, err
	}
	return total, nil
}

func (a *PublisherStorage) Query(q *Query) ([]*booklist.Publisher, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseSelectQuery, publisherFields.columns)
	if err != nil {
		return nil, err
	}

	rows, err := a.storage.db.Query(query,bindValues...)
	if err != nil {
		return nil, err
	}

	publishers := make([]*booklist.Publisher,0,16)

	defer rows.Close()
	for rows.Next() {
		publisher := &booklist.Publisher{}
		if err := publisherFields.scan(rows,publisher); err != nil {
			return nil, err
		}
		publishers = append(publishers, publisher)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return publishers, nil
}

func (a *PublisherStorage) ByID(sl []*booklist.Publisher) map[int]*booklist.Publisher {
	valuesByID := make(map[int]*booklist.Publisher)
	for _, value := range sl {
		valuesByID[value.ID] = value
	}
	return valuesByID
}