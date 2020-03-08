package pdf

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/sblinch/BookBrowser/booklist"
	"github.com/sblinch/BookBrowser/formats"
	"github.com/sblinch/BookBrowser/formatters"
	"github.com/pkg/errors"
)

type pdf struct {
	book *booklist.Book
}

func (e *pdf) Book() *booklist.Book {
	return e.book
}

func (e *pdf) HasCover() bool {
	return false
}

func (e *pdf) GetCover() (i io.ReadCloser, err error) {
	return nil, errors.New("no cover")
}

func load(filename string) (bi formats.BookInfo, ferr error) {
	p := &pdf{book: &booklist.Book{}}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, errors.Wrapf(err, "could not stat book")
	}
	p.book.FilePath = filename
	p.book.FileSize = fi.Size()
	p.book.ModTime = fi.ModTime()

	s := sha1.New()
	i, err := io.Copy(s, f)
	if err == nil && i != fi.Size() {
		err = errors.New("could not read whole file")
	}
	if err != nil {
		f.Close()
		return nil, errors.Wrap(err, "could not hash book")
	}
	p.book.Hash = fmt.Sprintf("%x", s.Sum(nil))

	f.Close()

	formatters.ApplyFilename(filename, p.book)

	pdf := NewPDFMeta()
	meta, err := pdf.ParseFile(filename)
	if meta != nil {
		if meta.Author != "" {
			p.book.Author = &booklist.Author{
				Name: meta.Author,
			}
		}
		if meta.Title != "" {
			p.book.Title = meta.Title
		}
	}

	debug.FreeOSMemory()

	return p, nil
}

func init() {
	formats.Register("pdf", load)
}
