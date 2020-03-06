package storage

import (
	"github.com/sblinch/BookBrowser/booklist"
	"database/sql"
	"time"
	"crypto/sha1"
	"fmt"
	"strings"
	)

// BookStorage provides database functionality for Book objects
type BookStorage struct {
	storage *Storage

	preparedInsert *sql.Stmt
	preparedUpdate *sql.Stmt

	baseSelectQuery string
	baseCountQuery string
}

// bookFields puts all of the code that maps database columns to struct fields in one place for ease of management when
// adding or removing fields.
//
// Each of these declarations each must remain in precisely the same order as the others (with the exception of the ID
// field which has special requirements noted below), and all declarations must be updated when fields are added or removed.
var bookFields = struct {
	table string
	columns []string

	insert func(stmt *sql.Stmt, book *booklist.Book) (sql.Result, error)
	update func(stmt *sql.Stmt, book *booklist.Book) (sql.Result, error)
	scan   func(rows *sql.Rows, book *booklist.Book, modtime *int64, pubDate *int64) error
}{
	table: "books",
	
	// id FIRST in columns
	columns: []string{"id", "pathname", "filesize", "filemtime", "hash", "hascover", "title", "description", "isbn", "publishdate", "authorid", "publisherid", "seriesid", "seriesindex"},
	scan: func(rows *sql.Rows, book *booklist.Book, modTime *int64, pubDate *int64) error {
		// id FIRST in scan
		return rows.Scan(&book.ID, &book.FilePath, &book.FileSize, modTime, &book.Hash, &book.HasCover, &book.Title, &book.Description, &book.ISBN, pubDate, &book.AuthorID, &book.PublisherID, &book.SeriesID, &book.SeriesIndex)
	},
	insert: func(stmt *sql.Stmt, book *booklist.Book) (sql.Result, error) {
		// id OMITTED in insert
		return stmt.Exec(book.FilePath, book.FileSize, book.ModTime.Unix(), book.Hash, book.HasCover, book.Title, book.Description, book.ISBN, book.PublishDate.Unix(), book.AuthorID, book.PublisherID, book.SeriesID, book.SeriesIndex)
	},
	update: func(stmt *sql.Stmt, book *booklist.Book) (sql.Result, error) {
		// id LAST in update
		return stmt.Exec(book.FilePath, book.FileSize, book.ModTime.Unix(), book.Hash, book.HasCover, book.Title, book.Description, book.ISBN, book.PublishDate.Unix(), book.AuthorID, book.PublisherID, book.SeriesID, book.SeriesIndex, book.ID)
	},
}

// Creates a new book storage object.
func NewBookStorage(s *Storage) (*BookStorage, error) {
	a := &BookStorage{
		storage: s,
	}
	var err error
	if a.preparedInsert, err = s.db.Prepare(buildInsertQuery(bookFields.table, bookFields.columns, false)); err != nil {
		return nil, err
	}

	if a.preparedUpdate, err = s.db.Prepare(buildUpdateQuery(bookFields.table, bookFields.columns)); err != nil {
		return nil, err
	}

	a.baseSelectQuery, a.baseCountQuery = buildSelectQuery(bookFields.table,bookFields.columns)

	return a, nil
}

// Saves all of this object's dependencies (authors, publishers, series) to the database if not already saved.
func (a *BookStorage) saveDeps(tx *sql.Tx, book *booklist.Book) error {
	if book.AuthorID == 0 && book.Author != nil && len(book.Author.Name) != 0 {
		if err := a.storage.Authors.SaveTx(tx,book.Author); err != nil {
			return fmt.Errorf("authors: %v",err)
		}
		book.AuthorID = book.Author.ID
	}
	if book.PublisherID == 0 && book.Publisher != nil && len(book.Publisher.Name) != 0 {
		if err := a.storage.Publishers.SaveTx(tx,book.Publisher); err != nil {
			return fmt.Errorf("publishers: %v",err)
		}
		book.PublisherID = book.Publisher.ID
	}
	if book.SeriesID == 0 && book.Series != nil && len(book.Series.Name) != 0 {
		if err := a.storage.Series.SaveTx(tx,book.Series); err != nil {
			return fmt.Errorf("series: %v",err)
		}
		book.SeriesID = book.Series.ID
	}
	return nil
}

// Saves one or more books and any dependencies (authors, etc) using the specified transaction.
func (a *BookStorage) SaveTx(tx *sql.Tx, books ... *booklist.Book) error {
	insertStmt := tx.Stmt(a.preparedInsert)
	updateStmt := tx.Stmt(a.preparedUpdate)

	var (
		res sql.Result
		insertID int64
		err error
	)
	for _, book := range books {
		if err = a.saveDeps(tx,book); err != nil {
			return fmt.Errorf("books, deps: %v",err)
		}

		if book.ID > 0 {
			if _, err = bookFields.update(updateStmt, book); err != nil {
				return fmt.Errorf("books, update: %v",err)
			}

		} else {
			res, err = bookFields.insert(insertStmt, book)
			if err == nil {
				insertID, err = res.LastInsertId()
				if err == nil {
					book.ID = int(insertID)
				}
			}
			if err != nil {
				return fmt.Errorf("books, insert: %v",err)
			}
		}
	}

	return nil
}

// Saves one or more books and any dependencies (authors, etc).
func (a *BookStorage) Save(books... *booklist.Book) error {
	tx, err := a.storage.db.Begin()
	if err != nil {
		return err
	}
	err = a.SaveTx(tx,books...)

	if err == nil {
		err = tx.Commit()
	} else {
		tx.Rollback()
	}

	return err
}

// Parses SQL result rows and creates a slice of Books.
func (a *BookStorage) parseRows(rows *sql.Rows, deps bool) ([]*booklist.Book, error) {
	books := make([]*booklist.Book, 0, 16)

	var (
		authorIDs,
		publisherIDs,
		seriesIDs map[int]struct{}
	)
	if deps {
		authorIDs, publisherIDs, seriesIDs = make(map[int]struct{}), make(map[int]struct{}), make(map[int]struct{})
	}

	var pubDate int64
	var modTime int64
	defer rows.Close()
	for rows.Next() {
		book := &booklist.Book{}
		if err := bookFields.scan(rows, book, &modTime, &pubDate); err != nil {
			return nil, err
		}
		book.PublishDate = time.Unix(pubDate, 0)
		book.ModTime = time.Unix(modTime,0)
		books = append(books, book)
		if deps {
			if _, exists := seriesIDs[book.SeriesID]; !exists {
				seriesIDs[book.SeriesID] = struct{}{}
			}
			if _, exists := authorIDs[book.AuthorID]; !exists {
				authorIDs[book.AuthorID] = struct{}{}
			}
			if _, exists := publisherIDs[book.PublisherID]; !exists {
				publisherIDs[book.PublisherID] = struct{}{}
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if deps {
		if err := a.loadBookDeps(books, authorIDs, seriesIDs, publisherIDs); err != nil {
			return nil, err
		}
	}

	return books, nil
}

// Provides common query logic used by both Query() and QueryDeps().
func (a *BookStorage) query(q *Query, deps bool) ([]*booklist.Book, error) {
	query, bindValues, err := q.buildSelect(a.baseSelectQuery,bookFields.columns)
	if err != nil {
		return nil, err
	}
	rows, err := a.storage.db.Query(query, bindValues...)
	if err != nil {
		return nil, err
	}

	return a.parseRows(rows, deps)
}

// Queries the database for one or more books.
func (a *BookStorage) Query(q *Query) ([]*booklist.Book, error) {
	return a.query(q, false)
}

// Queries the database for one or more books, and populates each resulting Book with its author, publisher, etc. records.
func (a *BookStorage) QueryDeps(q *Query) ([]*booklist.Book, error) {
	return a.query(q, true)
}

func (a *BookStorage) Count(q *Query) (int, error) {
	// specify columns explicitly (instead of *) to make sure Scan() encounters them in precisely the expected order
	query, bindValues, err := q.buildSelect(a.baseCountQuery,bookFields.columns)
	if err != nil {
		return -1, fmt.Errorf("count, building query: %v",err)
	}

	total := 0
	row := a.storage.db.QueryRow(query,bindValues...)
	if err := row.Scan(&total); err != nil {
		return -1, fmt.Errorf("count, scan: %v",err)
	}
	return total, nil
}


// Provides common query logic used by both QueryKeyword and CountKeyword
func (a *BookStorage) queryKeyword(keyword string, selectColumns []string, q *Query) (*sql.Rows, error) {
	keyword = "%" + keyword + "%"
	columnList := strings.Join(selectColumns, ",")
	baseQuery := fmt.Sprintf(`
SELECT DISTINCT %s
  FROM books
 WHERE books.authorid IN (SELECT id FROM authors WHERE name LIKE ?)
    OR books.seriesid IN (SELECT id FROM series WHERE series.name LIKE ?)
    OR books.title LIKE ?
`,columnList)

	query, bindValues, err := q.buildSelect(baseQuery,bookFields.columns)
	if err != nil {
		return nil, err
	}
	bindValues = append([]interface{}{ keyword, keyword, keyword },bindValues...)

	return a.storage.db.Query(query, bindValues...)
}

// Queries the database for books whose title, author name, or series name matches the given keyword.
func (a *BookStorage) QueryKeyword(keyword string, q *Query) ([]*booklist.Book, error) {
	rows, err := a.queryKeyword(keyword,getColumnsWithTable(bookFields.table,bookFields.columns),q)
	if err != nil {
		return nil, err
	}
	return a.parseRows(rows, true)
}

// Counts the number of books whose title, author name, or series name match the given keyword.
func (a *BookStorage) CountKeyword(keyword string, q *Query) (int, error) {
	rows, err := a.queryKeyword(keyword,[]string{"COUNT(DISTINCT books.id) AS total"},q)
	if err != nil {
		return 0, err
	}
	if !rows.Next() {
		return 0, nil
	}
	total := 0
	if err := rows.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// Converts an int-keyed struct{} map (used for efficiency/ease of readability in maintaining a list of unique ints) into an int slice.
func mapKeys(m map[int]struct{}) []int {
	r := make([]int, 0, len(m))
	for k, _ := range m {
		r = append(r, k)
	}
	return r
}

// loadBookDeps loads all dependencies (author, publisher, series) into each specified book
func (a *BookStorage) loadBookDeps(books []*booklist.Book, authoridmap map[int]struct{}, seriesidmap map[int]struct{}, publisheridmap map[int]struct{}) error {
	authorList, err := a.storage.Authors.Query(NewQuery().In("id", mapKeys(authoridmap)))
	if err != nil {
		return err
	}
	authorsById := a.storage.Authors.ByID(authorList)

	seriesList, err := a.storage.Series.Query(NewQuery().In("id", mapKeys(seriesidmap)))
	if err != nil {
		return err
	}
	seriesById := a.storage.Series.ByID(seriesList)

	publisherList, err := a.storage.Publishers.Query(NewQuery().In("id", mapKeys(publisheridmap)))
	if err != nil {
		return err
	}
	publishersById := a.storage.Publishers.ByID(publisherList)

	for _, book := range books {
		if author, exists := authorsById[book.AuthorID]; exists {
			book.Author = author
		} else {
			book.Author = &booklist.Author{}
		}
		if series, exists := seriesById[book.SeriesID]; exists {
			book.Series = series
		} else {
			book.Series = &booklist.Series{}
		}
		if publisher, exists := publishersById[book.PublisherID]; exists {
			book.Publisher = publisher
		} else {
			book.Publisher = &booklist.Publisher{}
		}
	}

	return nil
}

// Holds file metadata to help determine if an ebook file has been "seen" by the indexer before.
type BookSeen struct {
	FileSize int64
	ModTime  int64
}

// Returns a map of file size/time information for all books on file, keyed by filename hash
func (a *BookStorage) GetSeen() (map[string]BookSeen, error) {
	rows, err := a.storage.db.Query("SELECT pathname,filesize,filemtime FROM "+bookFields.table)
	if err != nil {
		return nil, err
	}

	seenList := make(map[string]BookSeen)
	filename := ""
	defer rows.Close()
	for rows.Next() {
		seen := BookSeen{}
		if err := rows.Scan(&filename, &seen.FileSize, &seen.ModTime); err != nil {
			return nil, err
		}
		filenameHash := fmt.Sprintf("%x", sha1.Sum([]byte(filename)))
		seenList[filenameHash] = seen
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return seenList, nil
}
