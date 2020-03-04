package storage

import (
	"github.com/sblinch/BookBrowser/booklist"
	"database/sql"
	"fmt"
)

type SeriesStorage struct {
	storage *Storage

	preparedInsert *sql.Stmt
	preparedUpdate *sql.Stmt
	preparedSelectID *sql.Stmt

	baseSelectQuery string
	baseCountQuery string
}

// seriesFields puts all of the code that maps database columns to struct fields in one place for ease of management when
// adding or removing fields.
//
// Each of these declarations each must remain in precisely the same order as the others (with the exception of the ID
// field which has special requirements noted below), and all declarations must be updated when fields are added or removed.
var seriesFields = struct {
	table string
	columns []string

	insert func(stmt *sql.Stmt, series *booklist.Series) (sql.Result, error)
	update func(stmt *sql.Stmt, series *booklist.Series) (sql.Result, error)
	scan   func(rows *sql.Rows, series *booklist.Series) error
}{
	table: "series",

	// id FIRST in columns
	columns: []string{"id", "name"},
	scan: func(rows *sql.Rows, series *booklist.Series) error {
		// id FIRST in scan
		return rows.Scan(&series.ID, &series.Name)
	},
	insert: func(stmt *sql.Stmt, series *booklist.Series) (sql.Result, error) {
		// id OMITTED in insert
		return stmt.Exec(series.Name)
	},
	update: func(stmt *sql.Stmt, series *booklist.Series) (sql.Result, error) {
		// id LAST in update
		return stmt.Exec(series.Name, series.ID)
	},
}


func NewSeriesStorage(s *Storage) (*SeriesStorage, error) {
	a := &SeriesStorage{
		storage: s,
	}

	var err error
	if a.preparedInsert, err = s.db.Prepare(buildInsertQuery(seriesFields.table,seriesFields.columns, true)); err != nil {
		return nil, err
	}
	if a.preparedUpdate, err = s.db.Prepare(buildUpdateQuery(seriesFields.table,seriesFields.columns)); err != nil {
		return nil, err
	}

	q, _ := buildSelectQuery(seriesFields.table,[]string{"id"})
	if a.preparedSelectID, err = s.db.Prepare(q+ " WHERE name=?"); err != nil {
		return nil, err
	}

	a.baseSelectQuery, a.baseCountQuery = buildSelectQuery(seriesFields.table,seriesFields.columns)

	return a, nil
}

func (a *SeriesStorage) Save(seriesList ... *booklist.Series) error {
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
	for _, series := range seriesList {
		if series.ID > 0 {
			if _, err = seriesFields.update(updateStmt,series); err != nil {
				return fmt.Errorf("series, update: %v",err)
			}

		} else {
			if err := selectIDStmt.QueryRow(series.Name).Scan(&existingID); err == nil && existingID > 0 {
				series.ID = existingID
				continue
			}

			res, err = seriesFields.insert(insertStmt,series)
			if err == nil {
				insertID, err = res.LastInsertId()
				if err == nil {
					series.ID = int(insertID)
				}
			}
			if err != nil {
				return fmt.Errorf("series, insert: %v",err)
			}
		}
	}
	if err = commit(); err != nil {
		return fmt.Errorf("series, commit: %v",err)
	}
	tx = nil
	return nil
}

func (a *SeriesStorage) Count(q *Query) (int, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseCountQuery,seriesFields.columns)
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

func (a *SeriesStorage) Query(q *Query) ([]*booklist.Series, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseSelectQuery,seriesFields.columns)
	if err != nil {
		return nil, err
	}

	rows, err := a.storage.db.Query(query,bindValues...)
	if err != nil {
		return nil, err
	}

	seriesList := make([]*booklist.Series,0,16)

	defer rows.Close()
	for rows.Next() {
		series := &booklist.Series{}
		if err := seriesFields.scan(rows,series); err != nil {
			return nil, err
		}
		seriesList = append(seriesList, series)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return seriesList, nil
}

func (a *SeriesStorage) ByID(sl []*booklist.Series) map[int]*booklist.Series {
	valuesByID := make(map[int]*booklist.Series)
	for _, value := range sl {
		valuesByID[value.ID] = value
	}
	return valuesByID
}