package formatters

import (
	"testing"
	"github.com/sblinch/BookBrowser/booklist"
	)

/*
func isLowerAlphaChar(c byte) bool {
func isAlphaChar(c byte) bool {
func isWordChar(c byte) bool {
func ucWords(s string) string {
func ucFirst(s string) string {
}

 */
func TestIsLowerAlphaChar(t *testing.T) {
	tests := map[byte]bool {
		'a': true,
		'z': true,
		'C': false,
		'9': false,
		'-': false,
		' ': false,
	}
	for input, expected := range tests {
		res := isLowerAlphaChar(input)
		if res != expected {
			t.Fatalf("for %c:\nexpected: %v\n     saw: %v",input,expected,res)
		}
	}
}

func TestIsAlphaChar(t *testing.T) {
	tests := map[byte]bool {
		'a': true,
		'z': true,
		'A': true,
		'Z': true,
		'9': false,
		'-': false,
		' ': false,
	}
	for input, expected := range tests {
		res := isAlphaChar(input)
		if res != expected {
			t.Fatalf("for %c:\nexpected: %v\n     saw: %v",input,expected,res)
		}
	}
}


func TestIsWordChar(t *testing.T) {
	tests := map[byte]bool {
		'a': true,
		'z': true,
		'A': true,
		'Z': true,
		'9': true,
		'-': false,
		' ': false,
	}
	for input, expected := range tests {
		res := isWordChar(input)
		if res != expected {
			t.Fatalf("for %c:\nexpected: %v\n     saw: %v",input,expected,res)
		}
	}
}

func TestUcWords(t *testing.T) {
	tests := map[string]string {
		"test": "Test",
		"Test": "Test",
		"test test": "Test Test",
		"Test test": "Test Test",
		"test Test": "Test Test",
		"0test test": "0test Test",
		" test test": " Test Test",
	}
	for input, expected := range tests {
		res := ucWords(input)
		if res != expected {
			t.Fatalf("for %s:\nexpected: %v\n     saw: %v",input,expected,res)
		}
	}
}

func TestUcFirst(t *testing.T) {
	tests := map[string]string {
		"test": "Test",
		"Test": "Test",
		"test test": "Test test",
		"Test test": "Test test",
		"test Test": "Test Test",
		"0test test": "0test test",
		" test test": " test test",
	}
	for input, expected := range tests {
		res := ucFirst(input)
		if res != expected {
			t.Fatalf("for %s:\nexpected: %v\n     saw: %v",input,expected,res)
		}
	}
}

func TestStripAuthor(t *testing.T) {
	tests := []struct{
		title string
		author string
		expected string
	}{
		{"This is a title","John Smith","This is a title"},
		{"John Smith - I can't feel my legs","John Smith","I can't feel my legs"},
		{"Smith, John - I can't feel my legs","John Smith","I can't feel my legs"},
		{"Smith, Johnson - I can't feel my legs","John Smith","Smith, Johnson - I can't feel my legs"},
		{"John Smitherson - I can't feel my legs","John Smith","John Smitherson - I can't feel my legs"},

		{"I can't feel my legs - John Smith","John Smith","I can't feel my legs"},
		{"I can't feel my legs - Smith, John","John Smith","I can't feel my legs"},
		{"I can't feel my legs - Smith, Johnson","John Smith","I can't feel my legs - Smith, Johnson"},
		{"I can't feel my legs - John Smitherson","John Smith","I can't feel my legs - John Smitherson"},
	}

	f := BookTitleFormatters["stripauthor"]
	for _, test := range tests {
		res := f(test.title,&booklist.Book{Author: &booklist.Author{Name: test.author}})
		if res != test.expected {
			t.Fatalf("for %s:\nexpected: %v\n     saw: %v",test.title,test.expected,res)
		}
	}

}