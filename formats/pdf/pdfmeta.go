package pdf

import (
	"os"
	"io"
	"io/ioutil"
	"github.com/beevik/etree"
	"fmt"
	"github.com/sblinch/BookBrowser/util"
)

type Metadata struct {
	Author string
	Title  string
}

type PDFMeta struct {
	buf []byte
}

func NewPDFMeta() *PDFMeta {
	return &PDFMeta{
		buf: make([]byte, 10240),
	}
}

func (p *PDFMeta) discard(r io.Reader) (int, error) {
	var (
		discarded int
		n         int
		err       error
	)
	for {
		n, err = r.Read(p.buf)
		discarded += n
		if err != nil {
			break
		}
	}
	return discarded, err
}

func (p *PDFMeta) parseXMP(raw []byte) (*Metadata, error) {
	xmp := etree.NewDocument()
	if err := xmp.ReadFromBytes(raw); err != nil {
		return nil, err
	}

	var (
		err error
		m   *Metadata
	)
	for _, d := range xmp.FindElements("//Description") {
		m, err = p.parseDescription(d)
		if m != nil {
			return m, nil
		}

	}
	return nil, err
}

func (p *PDFMeta) Parse(f io.ReadSeeker) (*Metadata, error) {
	r := util.NewReaderUntil(f, []byte("<?xpacket begin"))
	_, err := p.discard(r)
	if err != io.EOF {
		return nil, err
	}
	f.Seek(-int64(r.KeywordOffset), io.SeekCurrent)

	r = util.NewReaderUntil(f, []byte("<?xpacket end"))
	xmp, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(xmp) == 0 {
		return nil, nil
	}

	return p.parseXMP(xmp)
}

func (p *PDFMeta) ParseFile(filename string) (*Metadata, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return p.Parse(f)
}

func (p *PDFMeta) parseDescription(d *etree.Element) (m *Metadata, err error) {
	m = &Metadata{}

	attrFormat := d.SelectAttrValue("format", "")
	if attrFormat != "" && attrFormat != "application/pdf" {
		err = fmt.Errorf("description is of type %s", attrFormat)
		return
	}

	for _, e := range d.FindElements("format") {
		if e.Text() != "application/pdf" {
			err = fmt.Errorf("description is of type %s", e.Text())
			return
		}
	}

	for _, e := range d.FindElements("title/Alt/li") {
		m.Title = e.Text()
		if m.Title != "" {
			break
		}
	}

	for _, e := range d.FindElements("creator/Seq/li") {
		m.Author = e.Text()
		if m.Author != "" {
			break
		}
	}

	if m.Author == "" && m.Title == "" {
		m = nil
	}

	return
}
