package util

import (
	"io"
	"bytes"
)

// SectionReader reads from the underlying Reader until the specified keyword is encountered in the input
// stream; all bytes up to (but not including) the keyword are returned, followed by io.EOF
type ReaderUntil struct {
	// the underlying reader
	R io.Reader
	// once a keyword is found, the underlying reader has seeked this many bytes past the keyword
	KeywordOffset int

	keyword []byte

	keywordLen  int
	heldOver    []byte
	heldOverLen int
}

func (r *ReaderUntil) Read(p []byte) (n int, err error) {
	if r.keyword == nil {
		return 0, io.EOF
	}

	copy(p, r.heldOver[0:r.heldOverLen])
	n, err = r.R.Read(p[r.heldOverLen:])
	n += r.heldOverLen

	i := bytes.Index(p, r.keyword)
	if i != -1 {
		r.KeywordOffset = n - i

		n = i
		p = p[0:n]
		err = io.EOF
		r.keyword = nil
		r.heldOverLen = 0

	} else if (n-r.heldOverLen) > r.keywordLen && err == nil {
		copy(r.heldOver, p[n-r.keywordLen:])
		r.heldOverLen = r.keywordLen
		n -= r.keywordLen
	} else {
		r.heldOverLen = 0
	}

	return
}

func NewReaderUntil(rdr io.Reader, keyword []byte) *ReaderUntil {
	return &ReaderUntil{rdr, 0, keyword, len(keyword), make([]byte, len(keyword)), 0}
}
