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
	"github.com/sblinch/BookBrowser/formatters"
	"sync"
	"math/rand"
	"time"
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

// Use this many goroutines to concurrently read each ebook and create its cover/thumbnail files;
// the default value of 6 is arbitrary but was the lowest value that improved import time by a
// measurable amount for the author.
const importConcurrency = 6

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

	if i.Verbose {
		log.Printf("Scanning directories")
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

	indexChan := make(chan string,8)
	errorChan := make(chan error,8)
	errorsDone := make(chan struct{})

	go func() {
		for err := range errorChan {
			errs = append(errs, err)
		}
		close(errorsDone)
	}()

	importerGroup := sync.WaitGroup{}
	for n := 0; n<importConcurrency; n++ {
		importerGroup.Add(1)
		go func() {
			defer importerGroup.Done()

			// vary the transaction size by +/- 20% for each goroutine so that they don't all try to commit their
			// transactions at the same time
			txnSize := insertTransactionSize * 8/10 + rand.Int() % (insertTransactionSize * 4/10)

			newBooks := make([]*booklist.Book, 0, txnSize)
			for filepath := range indexChan {
				if i.Verbose {
					log.Printf("Indexing %s", filepath)
				}

				book, err := i.getBook(filepath)
				if err != nil {
					err = errors.Wrapf(err, "error reading book '%s'", filepath)
					errorChan <- err
					if i.Verbose {
						log.Printf("--> Error: %v", err)
					}
					continue
				} else {
					newBooks = append(newBooks, book)
					if len(newBooks) == cap(newBooks) {
						if err := i.storage.Books.Save(newBooks...); err != nil {
							errorChan <- err
							log.Printf("Fatal error: %v", err)
							return
						}

						newBooks = make([]*booklist.Book, 0, txnSize)
					}
				}
			}
			if len(newBooks) > 0 {
				i.storage.Books.Save(newBooks...)
			}
		}()
	}


	startTime := time.Now()

	for fi, filepath := range filenames {
		stat, err := os.Stat(filepath)
		if err != nil {
			errorChan <- errors.Wrapf(err, "cannot stat file '%s'", filepath)
			if i.Verbose {
				log.Printf("--> Error: %v", errs[len(errs)-1])
			}
			continue
		}

		filenameHash := fmt.Sprintf("%x", sha1.Sum([]byte(filepath)))
		if existing, exists := seen[filenameHash]; exists && existing.ModTime == stat.ModTime().Unix() && existing.FileSize == stat.Size() {
			if i.Verbose {
				log.Printf("Already seen %s; not reindexing",filepath)
			}
		} else {
			indexChan <- filepath
		}

		i.Progress = float64(fi+1) / float64(len(filenames))
	}
	close(indexChan)
	importerGroup.Wait()

	close(errorChan)
	<-errorsDone

	endTime := time.Now()

	if i.Verbose {
		log.Printf("Completed indexing in %v",endTime.Sub(startTime))
	}

	return errs, nil
}

// getBook loads the metadata for an ebook and prepares its cover images.
func (i *Indexer) getBook(filename string) (*booklist.Book, error) {
	bi, err := formats.Load(filename)
	if err != nil {
		return nil, errors.Wrap(err, "error loading book")
	}

	b := bi.Book()
	formatters.Apply(b)
	b.HasCover = false
	if i.datapath != nil && bi.HasCover() {
		coverpath := filepath.Join(*i.datapath, fmt.Sprintf("%s.jpg", b.Hash))
		thumbpath := filepath.Join(*i.datapath, fmt.Sprintf("%s_thumb.jpg", b.Hash))

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
	}

	return b, nil
}
