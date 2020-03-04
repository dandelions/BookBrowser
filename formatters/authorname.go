package formatters

import (
	"strings"
	"github.com/sblinch/BookBrowser/booklist"
)

var AuthorNameFormatters = map[string]StringFormatter{

	// removes all leading/trailing whitespace
	"trimspace": func(name string, book *booklist.Book) string {
		return strings.TrimSpace(name)
	},

	// converts Last, First into First Last
	"nameorder": func(name string, book *booklist.Book) string {
		names := strings.Split(name,", ")
		if len(names) == 2 {
			return names[1] + " " + names[0]
		} else {
			return name
		}
	},
	"case": func(name string, book *booklist.Book) string {
		if name == strings.ToLower(name) {
			name = ucWords(name)
		}
		return name
	},


}

var EnabledAuthorNameFormatters = []string{
	"trimspace","nameorder","case",
}
