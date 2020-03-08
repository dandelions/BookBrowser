package formatters

import (
	"strings"
	"github.com/sblinch/BookBrowser/booklist"
)

func isLowerAlphaChar(c byte) bool {
	return (c >= 'a' && c <= 'z')
}
func isAlphaChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
func ucWords(s string) string {
	b := []byte(s)
	uc := true
	for i, c := range b {
		if uc {
			if isAlphaChar(c) {
				uc = false
				if isLowerAlphaChar(c) {
					b[i] = c - 32
				}
			} else if c != ' '{
				uc = false
			}

		} else if c == ' ' {
			uc = true
		}
	}
	return string(b)
}

func ucFirst(s string) string {
	if len(s) > 0 && isLowerAlphaChar(s[0]) {
		b := []byte(s)
		b[0] = b[0] - 32
		s = string(b)
	}
	return s
}

var BookTitleFormatters = map[string]StringFormatter{
	// removes all leading/trailing whitespace
	"trimspace": func(name string, book *booklist.Book) string {
		return strings.TrimSpace(name)
	},

	// converts "Author Name - Title" to "Title"
	"stripauthor": func(name string, book *booklist.Book) string {
		if book.Author == nil || len(name) < len(book.Author.Name)+1 {
			return name
		}
		if strings.HasPrefix(name, book.Author.Name) {
			c := name[len(book.Author.Name)]
			if isWordChar(c) {
				name = strings.TrimPrefix(name, book.Author.Name)
				for len(name) > 0 && isWordChar(name[0]) {
					name = name[1:]
				}
			}
		}
		return name
	},

	// converts "all-lowercase" or "Just the first letter capitalized" titles to "All Words Capitalized" titles
	"case": func(name string, book *booklist.Book) string {
		ln := strings.ToLower(name)
		if name == strings.ToLower(ln) || name == ucFirst(ln) {
			name = ucWords(name)
		}
		return name
	},

	// removed double-quotes around titles
	"noquotes": func(name string, book *booklist.Book) string {
		if len(name) > 2 && name[0] == '"' && name[len(name)-1] == '"' {
			name = name[1 : len(name)-2]
		}
		return name
	},

	// changes "The Book Name" to "Book Name, The"
	"thelast": func(name string, book *booklist.Book) string {
		if len(name) > 4 && strings.ToLower(name[0:4]) == "the " {
			name = name[4:] + ", The"
		}
		return name
	},
}

var EnabledBookTitleFormatters = []string{
	"trimspace", "stripauthor", "case", "noquotes", "thelast",
}
