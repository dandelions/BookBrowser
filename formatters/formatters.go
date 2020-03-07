package formatters

import "github.com/sblinch/BookBrowser/booklist"

type StringFormatter func(name string, book *booklist.Book) string

func Apply(b *booklist.Book) {
	for _, n := range EnabledBookTitleFormatters {
		title := b.Title
		if formatter, exists := BookTitleFormatters[n]; exists {
			title = formatter(title, b)
		}
		b.Title = title
	}

	if b.Author != nil {
		name := b.Author.Name
		for _, n := range EnabledAuthorNameFormatters {
			if formatter, exists := AuthorNameFormatters[n]; exists {
				name = formatter(name, b)
			}
		}
		b.Author.Name = name
	}
}

func ApplyFilename(filename string, b *booklist.Book) {
	for _, n := range EnabledFilenameFormatters {
		if formatter, exists := FilenameFormatters[n]; exists {
			formatter(filename,b)
		}
	}

}