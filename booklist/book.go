package booklist

import (
	"path/filepath"
	"strings"
	"time"
)

type Book struct {
	ID       int
	Hash     string
	FilePath string
	FileSize int64
	ModTime  time.Time

	HasCover    bool
	Title       string
	Description string
	SeriesIndex float64
	ISBN        string
	PublishDate time.Time

	SeriesID    int
	AuthorID    int
	PublisherID int

	Author    *Author
	Series    *Series
	Publisher *Publisher
}

func (b *Book) FileType() string {
	return strings.Replace(strings.ToLower(filepath.Ext(b.FilePath)), ".", "", -1)
}
