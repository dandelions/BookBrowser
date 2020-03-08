package formatters

import (
	"testing"
	"reflect"
	"strings"
)

func TestUniqueStrings(t *testing.T) {
	tests := []struct{
		in []string
		out []string
	}{
		{ []string{"one", "two", "three", "one"}, []string{"one","two","three"} },
		{ []string{"one", "two", "one", "three"}, []string{"one","two","three"} },
		{ []string{"one", "one", "two", "three"}, []string{"one","two","three"} },
		{ []string{"one", "one", "two", "three","one"}, []string{"one","two","three"} },
	}
	for _, test := range tests {
		input := strings.Join(test.in,",")
		res := uniqueStrings(test.in)
		if !reflect.DeepEqual(res,test.out) {
			t.Fatalf("for %s:\nexpected: %v\n     saw: %v",input,strings.Join(test.out,","),strings.Join(res,","))
		}
	}


}
