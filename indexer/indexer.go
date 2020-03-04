package indexer

import (
	"fmt"
	"log"
	"os"
		"image/jpeg"
	"path/filepath"
	"crypto/sha1"

	"github.com/sblinch/BookBrowser/booklist"
	"github.com/sblinch/BookBrowser/formats"
	"github.com/sblinch/BookBrowser/storage"

	"github.com/mattn/go-zglob"
	"github.com/nfnt/resize"
	"github.com/pkg/errors"
	"sync/atomic"
)

// An Indexer walks filesystem path(s) and imports book data into the index database.
type Indexer struct {
	Verbose  bool
	Progress float64
	storage  *storage.Storage
	datapath *string
	paths    []string
	exts     []string
	booklist booklist.BookList

	indexingActive uint32
}

// Creates a new Indexer.
func New(paths []string, storage *storage.Storage, datapath *string, exts []string) (*Indexer, error) {
	for i := range paths {
		p, err := filepath.Abs(paths[i])
		if err != nil {
			return nil, errors.Wrap(err, "error resolving path")
		}
		paths[i] = p
	}

	cp := (*string)(nil)
	if datapath != nil {
		p, err := filepath.Abs(*datapath)
		if err != nil {
			return nil, errors.Wrap(err, "error resolving cover path")
		}
		cp = &p
	}

	return &Indexer{paths: paths, storage: storage, datapath: cp, exts: exts}, nil
}

// Store this many books in memory before writing a batch to the database for improved performance
const insertTransactionSize = 64

// Refresh updates the index database.
func (i *Indexer) Refresh() ([]error, error) {
	errs := []error{}

	if !atomic.CompareAndSwapUint32(&i.indexingActive,0,1) {
		return errs, errors.New("indexing is already in progress")
	}
	defer atomic.StoreUint32(&i.indexingActive,0)

	if len(i.paths) < 1 {
		return errs, errors.New("no paths to index")
	}

	seen, err := i.storage.Books.GetSeen()
	if err != nil {
		return errs, err
	}

	filenames := []string{}
	for _, path := range i.paths {
		for _, ext := range i.exts {
			l, err := zglob.Glob(filepath.Join(path, "**", fmt.Sprintf("*.%s", ext)))
			if l != nil {
				filenames = append(filenames, l...)
			}
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "error scanning '%s' for type '%s'", path, ext))
				if i.Verbose {
					log.Printf("Error: %v", errs[len(errs)-1])
				}
			}
		}
	}

	defer func() {
		i.Progress = 0
	}()

	newBooks := make([]*booklist.Book, 0, insertTransactionSize)
	for fi, filepath := range filenames {
		if i.Verbose {
			log.Printf("Indexing %s", filepath)
		}

		stat, err := os.Stat(filepath)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "cannot stat file '%s'", filepath))
			if i.Verbose {
				log.Printf("--> Error: %v", errs[len(errs)-1])
			}
			continue
		}

		filenameHash := fmt.Sprintf("%x", sha1.Sum([]byte(filepath)))
		if existing, exists := seen[filenameHash]; exists && existing.ModTime == stat.ModTime().Unix() && existing.FileSize == stat.Size() {
			if i.Verbose {
				log.Printf("Already seen; not reindexing")
			}
		} else {
			book, err := i.getBook(filepath)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "error reading book '%s'", filepath))
				if i.Verbose {
					log.Printf("--> Error: %v", errs[len(errs)-1])
				}
				continue
			} else {
				seen[filenameHash] = storage.BookSeen{stat.Size(), stat.ModTime().Unix()}

				newBooks = append(newBooks, book)
				if len(newBooks) == cap(newBooks) {
					if err := i.storage.Books.Save(newBooks...); err != nil {
						log.Fatalf("Fatal error: %v",err)
					}
					newBooks = make([]*booklist.Book, 0, insertTransactionSize)
				}
			}
		}

		i.Progress = float64(fi+1) / float64(len(filenames))
	}

	if len(newBooks) > 0 {
		i.storage.Books.Save(newBooks...)
	}

	return errs, nil
}

// getBook loads the metadata for an ebook and prepares its cover images.
func (i *Indexer) getBook(filename string) (*booklist.Book, error) {
	// TODO: caching
	bi, err := formats.Load(filename)
	if err != nil {
		return nil, errors.Wrap(err, "error loading book")
	}

	b := bi.Book()
	b.HasCover = false
	if i.datapath != nil && bi.HasCover() {
		coverpath := filepath.Join(*i.datapath, fmt.Sprintf("%s.jpg", b.Hash))
		thumbpath := filepath.Join(*i.datapath, fmt.Sprintf("%s_thumb.jpg", b.Hash))

		log.Printf("has cover: %s %s",coverpath,thumbpath)

		_, err := os.Stat(coverpath)
		_, errt := os.Stat(thumbpath)
		if err != nil || errt != nil {
			i, err := bi.GetCover()
			if err != nil {
				return nil, errors.Wrap(err, "error getting cover")
			}

			f, err := os.Create(coverpath)
			if err != nil {
				return nil, errors.Wrap(err, "could not create cover file")
			}
			defer f.Close()

			err = jpeg.Encode(f, i, nil)
			if err != nil {
				os.Remove(coverpath)
				return nil, errors.Wrap(err, "could not write cover file")
			}

			ti := resize.Thumbnail(400, 400, i, resize.Bicubic)

			tf, err := os.Create(thumbpath)
			if err != nil {
				return nil, errors.Wrap(err, "could not create cover thumbnail file")
			}
			defer tf.Close()

			err = jpeg.Encode(tf, ti, nil)
			if err != nil {
				os.Remove(thumbpath)
				return nil, errors.Wrap(err, "could not write cover thumbnail file")
			}
		}

		b.HasCover = true
	} else {
		log.Printf("has no cover")
	}

	return b, nil
}
