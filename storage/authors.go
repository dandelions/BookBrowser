package storage

import (
	"github.com/sblinch/BookBrowser/booklist"
	"database/sql"
	"fmt"
	"github.com/sblinch/BookBrowser/util"
)

type AuthorStorage struct {
	storage *Storage

	preparedInsert *sql.Stmt
	preparedUpdate *sql.Stmt
	preparedSelectID *sql.Stmt

	baseSelectQuery string
	baseSelectIDQuery string
	baseCountQuery string
}

// authorFields puts all of the code that maps database columns to struct fields in one place for ease of management when
// adding or removing fields.
//
// Each of these declarations each must remain in precisely the same order as the others (with the exception of the ID
// field which has special requirements noted below), and all declarations must be updated when fields are added or removed.
var authorFields = struct {
	table string
	columns []string

	insert func(stmt *sql.Stmt, author *booklist.Author) (sql.Result, error)
	update func(stmt *sql.Stmt, author *booklist.Author) (sql.Result, error)
	scan   func(rows *sql.Rows, author *booklist.Author) error
}{
	table: "authors",
	
	// id FIRST in columns
	columns: []string{"id", "name", "sortname"},
	scan: func(rows *sql.Rows, author *booklist.Author) error {
		// id FIRST in scan
		return rows.Scan(&author.ID, &author.Name, &author.SortName)
	},
	insert: func(stmt *sql.Stmt, author *booklist.Author) (sql.Result, error) {
		// id OMITTED in insert
		return stmt.Exec(author.Name, author.SortName)
	},
	update: func(stmt *sql.Stmt, author *booklist.Author) (sql.Result, error) {
		// id LAST in update
		return stmt.Exec(author.Name, author.SortName, author.ID)
	},
}


func NewAuthorStorage(s *Storage) (*AuthorStorage, error) {
	a := &AuthorStorage{
		storage: s,
	}

	var err error
	if a.preparedInsert, err = s.db.Prepare(buildInsertQuery(authorFields.table,authorFields.columns, true)); err != nil {
		return nil, err
	}
	if a.preparedUpdate, err = s.db.Prepare(buildUpdateQuery(authorFields.table,authorFields.columns)); err != nil {
		return nil, err
	}

	q, _ := buildSelectQuery(authorFields.table,[]string{"id"})
	if a.preparedSelectID, err = s.db.Prepare(q+ " WHERE name=?"); err != nil {
		return nil, err
	}

	a.baseSelectQuery, a.baseCountQuery = buildSelectQuery(authorFields.table,authorFields.columns)

	return a, nil
}


func (a *AuthorStorage) SetSortName(author *booklist.Author) {
	if author == nil || author.Name == "" {
		author.SortName = ""
	} else {
		author.SortName = util.LastNameFirst(author.Name)
	}
}

// Saves one or more Authors using the specified transaction.
func (a *AuthorStorage) SaveTx(tx *sql.Tx, authors ... *booklist.Author) error {
	insertStmt := tx.Stmt(a.preparedInsert)
	updateStmt := tx.Stmt(a.preparedUpdate)
	selectIDStmt := tx.Stmt(a.preparedSelectID)

	var (
		res sql.Result
		insertID int64
		existingID int
		err error
	)
	for _, author := range authors {
		a.SetSortName(author)
		if author.ID > 0 {
			if _, err = authorFields.update(updateStmt,author); err != nil {
				return fmt.Errorf("authors, update: %v",err)
			}

		} else {
			if author.Name == "" {
				author.Name = "Unknown"
			}
			if err := selectIDStmt.QueryRow(author.Name).Scan(&existingID); err == nil && existingID > 0 {
				author.ID = existingID
				continue
			}

			res, err = authorFields.insert(insertStmt,author)
			if err == nil {
				insertID, err = res.LastInsertId()
				if err == nil {
					author.ID = int(insertID)
				}
			}
			if err != nil {
				return fmt.Errorf("authors, insert: %v",err)
			}
		}
	}

	return nil

}

// Saves one or more Authors.
func (a *AuthorStorage) Save(authors ... *booklist.Author) error {
	tx, err := a.storage.db.Begin()
	if err != nil {
		return err
	}
	err = a.SaveTx(tx,authors...)

	if err == nil {
		err = tx.Commit()
	} else {
		tx.Rollback()
	}

	return err
}

func (a *AuthorStorage) Count(q *Query) (int, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseCountQuery,authorFields.columns)
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

func (a *AuthorStorage) Query(q *Query) ([]*booklist.Author, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseSelectQuery,authorFields.columns)
	if err != nil {
		return nil, fmt.Errorf("authors, buildselect: %v",err)
	}
	rows, err := a.storage.db.Query(query,bindValues...)
	if err != nil {
		return nil,  fmt.Errorf("authors, query: %v",err)
	}

	authors := make([]*booklist.Author,0,16)

	defer rows.Close()
	for rows.Next() {
		author := &booklist.Author{}
		if err := authorFields.scan(rows,author); err != nil {
			return nil,  fmt.Errorf("authors, scan: %v",err)
		}
		authors = append(authors, author)
	}
	if err = rows.Close(); err != nil {
		return nil,  fmt.Errorf("authors, close: %v",err)
	}

	if err = rows.Err(); err != nil {
		return nil,  fmt.Errorf("authors, rows: %v",err)
	}

	return authors, nil
}

func (a *AuthorStorage) ByID(sl []*booklist.Author) map[int]*booklist.Author {
	valuesByID := make(map[int]*booklist.Author)
	for _, value := range sl {
		valuesByID[value.ID] = value
	}
	return valuesByID
}
