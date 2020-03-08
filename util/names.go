package util

import (
	"strings"
)

var surnameSuffixes = map[string]struct{}{
	"jr": {}, "jr.": {},
	"sr": {}, "sr.": {},
	"i":    {},
	"ii":   {},
	"iii":  {},
	"iv":   {},
	"v":    {},
	"vi":   {},
	"vii":  {},
	"viii": {},
	"ix":   {},
	"x":    {},
	"xi":   {},
	"xii":  {},
	"xiii": {},
	"xiv":  {},
	"xv":   {},
}
var surnamePrefixes = map[string]struct{}{
	"a":    {},
	"ab":   {},
	"af":   {},
	"ap":   {},
	"abu":  {},
	"aït":  {},
	"al":   {},
	"ālam": {}, "alam": {},
	"aust": {}, "austre": {},
	"bar":  {},
	"bath": {}, "bat": {},
	"ben": {}, "bin": {}, "ibn": {},
	"bet":   {},
	"bint":  {},
	"da":    {},
	"das":   {},
	"de":    {},
	"degli": {},
	"dele":  {}, "del": {},
	"della": {}, "de la": {},
	"der":   {},
	"di":    {},
	"dos":   {},
	"du":    {},
	"e":     {},
	"el":    {},
	"fetch": {}, "vetch": {},
	"fitz": {},
	"i":    {},
	"kil":  {}, "gil": {},
	"la":    {},
	"le":    {},
	"lille": {},
	"lu":    {},
	"mac":   {}, "mc": {}, "mck": {}, "mhic": {}, "mic": {},
	"mala":   {},
	"mellom": {}, "myljom": {},
	"na":  {},
	"ned": {}, "nedre": {},
	"neder": {},
	"nic":   {}, "ni": {},
	"nin":  {},
	"nord": {}, "norr": {},
	"ny": {},
	"o":  {}, "ua": {}, "ui": {},
	"opp": {}, "upp": {},
	"öfver": {}, "ofver": {},
	"ost": {}, "öst": {}, "öster": {}, "oster": {}, "øst": {}, "østre": {}, "ostre": {},
	"över": {}, "over": {},
	"øvste": {}, "øvre": {}, "øver": {}, "ovste": {}, "ovre": {},
	"öz": {}, "oz": {},
	"pour":  {},
	"stor":  {},
	"söder": {}, "soder": {},
	"ter":  {},
	"tre":  {},
	"van":  {},
	"väst": {}, "väster": {}, "vast": {}, "vaster": {},
	"verch": {}, "erch": {},
	"vest":  {},
	"vesle": {}, "vetle": {},
	"von": {},
	"zu":  {},
}

func LastNameFirst(name string) string {
	if name == "" {
		return ""
	}

	names := SplitAny(name, []string{";","&"," and "})
	for k, name := range names {
		name := strings.TrimSpace(name)

		pieces := strings.Split(name, " ")
		if len(pieces) == 1 {
			continue
		} else if len(pieces) == 2 {
			names[k] = pieces[1] + ", " + pieces[0]
		} else {
			surname := strings.TrimSuffix(pieces[len(pieces)-1], ",")
			pieces = pieces[0 : len(pieces)-1]

			if _, exists := surnameSuffixes[strings.ToLower(surname)]; exists {
				surname = strings.TrimSuffix(pieces[len(pieces)-1], ",") + " " + surname
				pieces = pieces[0 : len(pieces)-1]
			}

			possiblePrefix := strings.TrimSuffix(pieces[len(pieces)-1], ",")
			if _, exists := surnamePrefixes[strings.ToLower(possiblePrefix)]; exists {
				surname = possiblePrefix + " " + surname
				pieces = pieces[0 : len(pieces)-1]
			}

			names[k] = surname + ", " + strings.Join(pieces, " ")
		}
	}

	return strings.Join(names, "; ")
}

func SplitAny(s string, sep []string) []string {
	res := make([]string, 0, 8)

	p := make([]int, len(sep))
	for k, _ := range p {
		p[k] = -2
	}

	removed := 0
	piece := ""
	for {
		firstSepKey := -1
		firstIndex := -1
		for k, v := range sep {
			if p[k] == -1 {
				continue
			} else if p[k] == -2 {
				p[k] = strings.Index(s, v)
				if p[k] == -1 {
					continue
				}
			} else {
				p[k] -= removed
			}
			if firstIndex == -1 || p[k] < firstIndex {
				firstIndex = p[k]
				firstSepKey = k
			}
		}
		if firstIndex == -1 {
			break
		} else if firstIndex > 0 {
			piece = s[0:firstIndex]
		} else {
			piece = ""
		}
		res = append(res, piece)
		removed = firstIndex + len(sep[firstSepKey])
		s = s[removed:]
		if len(s) == 0 {
			break
		}
		p[firstSepKey] = -2
	}

	if len(s) > 0 {
		res = append(res, s)
	}

	return res
}
