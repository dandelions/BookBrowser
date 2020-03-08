package util

import (
	"testing"
	"reflect"
	"strings"
)

func TestLastNameFirst(t *testing.T) {
	tests := map[string]string{
		"Plato": "Plato",
		"John Smith": "Smith, John",
		"John Q. Smith": "Smith, John Q.",
		"John Smith, III": "Smith III, John",
		"John Smith III": "Smith III, John",
		"John Q. Smith, III": "Smith III, John Q.",
		"Dick van Dyke": "van Dyke, Dick",
		"Dara O Briain": "O Briain, Dara",
		"John Smith Jr": "Smith Jr, John",
		"John Smith, Jr.": "Smith Jr., John",
		"John Du Bois": "Du Bois, John",
	}

	for input, expected := range tests {
		res := LastNameFirst(input)
		if res != expected {
			t.Fatalf("for %s:\nexpected: %s\n     saw: %s",input,expected,res)
		}
	}
}

func TestSplitAny(t *testing.T) {
	tests := map[string][]string{
		"John Smith": {"John Smith"},
		"John Smith; Jimi Hendrix": {"John Smith"," Jimi Hendrix"},
		"John Smith and Jimi Hendrix": {"John Smith","Jimi Hendrix"},
		"John Smith & Jimi Hendrix": {"John Smith "," Jimi Hendrix"},
		"John Smith; Bill Clinton & Jimi Hendrix": {"John Smith"," Bill Clinton "," Jimi Hendrix"},
		"John Smith and Kiefer Sutherland; Bill Clinton & Jimi Hendrix": {"John Smith","Kiefer Sutherland"," Bill Clinton "," Jimi Hendrix"},
		"John Smith and Kiefer Sutherland and Bill Clinton and Jimi Hendrix": {"John Smith","Kiefer Sutherland","Bill Clinton","Jimi Hendrix"},
		"John Smith & Kiefer Sutherland and Bill Clinton & Jimi Hendrix": {"John Smith "," Kiefer Sutherland","Bill Clinton "," Jimi Hendrix"},
	}

	for input, expected := range tests {
		res := SplitAny(input,[]string{";","&"," and "})
		if !reflect.DeepEqual(res,expected) {
			t.Fatalf("for %s:\nexpected: %v\n     saw: %v",input,strings.Join(expected,"|"),strings.Join(res,"|"))
		}
	}

}