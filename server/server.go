package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"github.com/sblinch/BookBrowser/formats"
	"github.com/sblinch/BookBrowser/indexer"
	"github.com/sblinch/BookBrowser/public"
	//"github.com/geek1011/kepubify/kepub"
	"github.com/julienschmidt/httprouter"
	"github.com/unrolled/render"
	"github.com/sblinch/BookBrowser/storage"
)

// Server is a BookBrowser server.
type Server struct {
	Indexer  *indexer.Indexer
	BookDir  string
	DataDir  string
	NoCovers bool
	Addr     string
	Verbose  bool
	storage  *storage.Storage
	router   *httprouter.Router
	render   *render.Render
	version  string
}

// NewServer creates a new BookBrowser server. It will not index the books automatically.
func NewServer(addr string, stor *storage.Storage, bookdir, datadir, version string, verbose, nocovers bool) *Server {
	i, err := indexer.New([]string{bookdir}, stor, &datadir, formats.GetExts())
	if err != nil {
		panic(err)
	}
	i.Verbose = verbose

	if verbose {
		log.Printf("Supported formats: %s", strings.Join(formats.GetExts(), ", "))
	}

	s := &Server{
		Indexer:  i,
		BookDir:  bookdir,
		Addr:     addr,
		DataDir:  datadir,
		NoCovers: nocovers,
		Verbose:  verbose,
		storage:  stor,
		router:   httprouter.New(),
		version:  version,
	}

	s.initRender()
	s.initRouter()

	return s
}

// printLog runs log.Printf if verbose is true.
func (s *Server) printLog(format string, v ...interface{}) {
	if s.Verbose {
		log.Printf(format, v...)
	}
}

// RefreshBookIndex refreshes the book index
func (s *Server) RefreshBookIndex() error {
	errs, err := s.Indexer.Refresh()
	if err != nil {
		log.Printf("Error indexing: %s", err)
		return err
	}
	if len(errs) != 0 {
		if s.Verbose {
			log.Printf("Indexing finished with %v errors", len(errs))
		}
	} else {
		log.Printf("Indexing finished")
	}

	debug.FreeOSMemory()

	return nil
}

// Serve starts the BookBrowser server. It does not return unless there is an error.
func (s *Server) Serve() error {
	s.printLog("Serving on %s\n", s.Addr)
	err := http.ListenAndServe(s.Addr, s.router)
	if err != nil {
		return err
	}
	return nil
}

// initRender initializes the renderer for the BookBrowser server.
func (s *Server) initRender() {
	s.render = render.New(render.Options{
		Directory:  "templates",
		Asset:      public.Box.MustBytes,
		AssetNames: public.Box.List,
		Layout:     "base",
		Extensions: []string{".tmpl"},
		Funcs: []template.FuncMap{
			template.FuncMap{
				"ToUpper": strings.ToUpper,
				"raw": func(s string) template.HTML {
					return template.HTML(s)
				},
			},
		},
		IsDevelopment: false,
	})
}

// Allows cover images to be requested by the ebook hash, but converts the requested filename into the two-tier
// directory format we use internally
type CoverDir struct {
	http.Dir
}

func (d CoverDir) Open(name string) (http.File, error) {
	return d.Dir.Open(name[0:3] + "/" + name[3:])
}

// initRouter initializes the router for the BookBrowser server.
func (s *Server) initRouter() {
	s.router = httprouter.New()

	s.router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		http.Redirect(w, r, "/books/", http.StatusTemporaryRedirect)
	})

	s.router.GET("/random", s.handleRandom)

	s.router.GET("/search", s.handleSearch)
	s.router.GET("/search/authors", s.handleSearchAuthors)
	s.router.GET("/search/series", s.handleSearchSeries)

	s.router.GET("/api/indexer", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"indexing": %t, "progress": %f}`, s.Indexer.Progress != 0, s.Indexer.Progress)
	})

	s.router.GET("/books", s.handleBooks)
	s.router.GET("/books/:id", s.handleBook)

	s.router.GET("/authors", s.handleAuthors)
	s.router.GET("/authors/:id", s.handleAuthor)

	s.router.GET("/series", s.handleSeriess)
	s.router.GET("/series/:id", s.handleSeries)

	s.router.GET("/download", s.handleDownloads)
	s.router.GET("/download/:filename", s.handleDownload)

	s.router.GET("/static/*filepath", func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		http.FileServer(public.Box).ServeHTTP(w, req)
	})
	s.router.ServeFiles("/covers/*filepath", CoverDir{Dir: http.Dir(s.DataDir)})
}

func (s *Server) handleDownloads(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/html")
	var buf bytes.Buffer
	buf.WriteString(`
<!DOCTYPE html>
<html>
<head>
<title>BookBrowser</title>
<style>
a,
a:link,
a:visited {
display:  block;
white-space: nowrap;
text-overflow: ellipsis;
color: inherit;
text-decoration: none;
font-family: sans-serif;
padding: 5px 7px;
background:  #FAFAFA;
border-bottom: 1px solid #DDDDDD;
cursor: pointer;
}

a:hover,
a:active {
background: #EEEEEE;
}

html, body {
background: #FAFAFA;
margin: 0;
padding: 0;
}
</style>
</head>
<body>
	`)
	sbl, err := s.storage.Books.QueryDeps(storage.NewQuery().SortedBy("title", true))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "Error handling request")
		return
	}

	for _, b := range sbl {
		if b.Author.Name != "" && b.Series.Name != "" {
			buf.WriteString(fmt.Sprintf("<a href=\"/download/%d.%s\">%s - %s - %s (%v)</a>", b.ID, b.FileType(), b.Title, b.Author.Name, b.Series.Name, b.SeriesIndex))
		} else if b.Author.Name != "" && b.Series.Name == "" {
			buf.WriteString(fmt.Sprintf("<a href=\"/download/%d.%s\">%s - %s</a>", b.ID, b.FileType(), b.Title, b.Author.Name))
		} else if b.Author.Name == "" && b.Series.Name != "" {
			buf.WriteString(fmt.Sprintf("<a href=\"/download/%d.%s\">%s - %s (%v)</a>", b.ID, b.FileType(), b.Title, b.Series.Name, b.SeriesIndex))
		} else if b.Author.Name == "" && b.Series.Name == "" {
			buf.WriteString(fmt.Sprintf("<a href=\"/download/%d.%s\">%s</a>", b.ID, b.FileType(), b.Title))
		}
	}
	buf.WriteString(`
</body>
</html>
	`)
	io.WriteString(w, buf.String())
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	bid := p.ByName("filename")
	bid = strings.Replace(strings.Replace(bid, filepath.Ext(bid), "", 1), ".kepub", "", -1)
	iskepub := false
	/*
	if strings.HasSuffix(p.ByName("filename"), ".kepub.epub") {
		iskepub = true
	}
	*/

	bl, err := s.storage.Books.Query(storage.NewQuery().Filtered("id", bid, true))
	if err != nil || len(bl) == 0 {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, "Could not find book with id "+bid)
		return
	}

	b := bl[0]
	if !iskepub {
		rd, err := os.Open(b.FilePath)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "Error handling request")
			log.Printf("Error handling request for %s: %s\n", r.URL.Path, err)
			return
		}

		w.Header().Set("Content-Disposition", `attachment; filename="`+regexp.MustCompile("[[:^ascii:]]").ReplaceAllString(b.Title, "_")+`.`+b.FileType()+`"`)
		switch b.FileType() {
		case "epub":
			w.Header().Set("Content-Type", "application/epub+zip")
		case "pdf":
			w.Header().Set("Content-Type", "application/pdf")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		_, err = io.Copy(w, rd)
		rd.Close()
		if err != nil {
			log.Printf("Error handling request for %s: %s\n", r.URL.Path, err)
		}
	} /*else {
		if b.FileType() != "epub" {
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, "Not found")
			return
		}
		td, err := ioutil.TempDir("", "kepubify")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error handling request for %s: %s\n", r.URL.Path, err)
			io.WriteString(w, "Internal Server Error")
			return
		}
		defer os.RemoveAll(td)
		kepubf := filepath.Join(td, bid+".kepub.epub")
		err = (&kepub.Converter{}).Convert(b.FilePath, kepubf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error handling request for %s: %s\n", r.URL.Path, err)
			io.WriteString(w, "Internal Server Error - Error converting book")
			return
		}
		rd, err := os.Open(kepubf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "Error handling request")
			log.Printf("Error handling request for %s: %s\n", r.URL.Path, err)
			return
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+url.PathEscape(b.Title)+".kepub.epub")
		w.Header().Set("Content-Type", "application/epub+zip")
		_, err = io.Copy(w, rd)
		rd.Close()
		if err != nil {
			log.Printf("Error handling request for %s: %s\n", r.URL.Path, err)
		}
	}*/
}

func (s *Server) internalError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, "Error handling request")
	if err != nil {
		log.Println(err)
	}
}

func (s *Server) handleAuthors(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	query := storage.NewQuery().SortedBy("sortname", true)
	totalAuthors, err := s.storage.Authors.Count(query)
	if err != nil {
		s.internalError(w, err)
		return
	}

	pagination := NewPagination(r.URL.Query(), totalAuthors)
	al, err := s.storage.Authors.Query(query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit))
	if err != nil {
		s.internalError(w, err)
		return
	}

	s.render.HTML(w, http.StatusOK, "authors", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Authors",
		"ShowBar":          true,
		"ShowSearch":       false,
		"ShowAuthorSearch": true,
		"ShowSeriesSearch": false,
		"ShowViewSelector": true,
		"Title":            "Authors",
		"Authors":          al,
		"Pagination":       pagination,
	})
}

func parseUserSort(s string, defaultKey string, defaultAscending bool) (key string, ascending bool) {
	if len(s) == 0 {
		return defaultKey, defaultAscending
	}
	pieces := strings.SplitN(s, "-", 2)
	if len(pieces) != 2 || pieces[0] == "" {
		return defaultKey, defaultAscending
	} else {
		return pieces[0], pieces[1] == "asc"
	}
}
func (s *Server) handleAuthor(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	aid := p.ByName("id")
	authors, err := s.storage.Authors.Query(storage.NewQuery().Filtered("id", aid, true))
	if err != nil {
		s.internalError(w, fmt.Errorf("query-authors: %v", err))
		return
	}

	if len(authors) > 0 {
		author := authors[0]
		userSortKey, userSortAsc := parseUserSort(r.URL.Query().Get("sort"), "title", true)

		query := storage.NewQuery().Filtered("authorid", aid, true).SortedBy(userSortKey, userSortAsc)
		total, err := s.storage.Books.Count(query)
		if err != nil {
			s.internalError(w, fmt.Errorf("count-books: %v", err))
			return
		}

		pagination := NewPagination(r.URL.Query(), total)
		query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit)

		bl, err := s.storage.Books.QueryDeps(query)
		if err != nil {
			s.internalError(w, fmt.Errorf("query-books: %v", err))
			return
		}

		s.render.HTML(w, http.StatusOK, "author", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        author.Name,
			"ShowBar":          true,
			"ShowSearch":       false,
			"ShowAuthorSearch": false,
			"ShowSeriesSearch": false,
			"ShowViewSelector": true,
			"Title":            author.Name,
			"Books":            bl,
			"Pagination":       pagination,
		})
		return
	}

	s.render.HTML(w, http.StatusNotFound, "notfound", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Not Found",
		"ShowBar":          false,
		"ShowSearch":       false,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": false,
		"ShowViewSelector": false,
		"Title":            "Not Found",
		"Message":          "Author not found.",
	})
}

func (s *Server) handleSeriess(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := storage.NewQuery().SortedBy("name", true)
	total, err := s.storage.Series.Count(query)
	if err != nil {
		s.internalError(w, err)
		return
	}

	pagination := NewPagination(r.URL.Query(), total)
	query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit)

	seriess, err := s.storage.Series.Query(query)
	if err != nil {
		s.internalError(w, err)
		return
	}

	s.render.HTML(w, http.StatusOK, "seriess", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Series",
		"ShowBar":          true,
		"ShowSearch":       false,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": true,
		"ShowViewSelector": true,
		"Title":            "Series",
		"Series":           seriess,
		"Pagination":       pagination,
	})
}

func (s *Server) handleSeries(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	sid := p.ByName("id")

	seriess, err := s.storage.Series.Query(storage.NewQuery().Filtered("id", sid, true))
	if err != nil {
		s.internalError(w, err)
		return
	}

	if len(seriess) > 0 {
		series := seriess[0]

		query := storage.NewQuery().Filtered("seriesid", sid, true).SortedBy("seriesindex", true)
		total, err := s.storage.Books.Count(query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		pagination := NewPagination(r.URL.Query(), total)
		query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit)

		bl, err := s.storage.Books.QueryDeps(query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		s.render.HTML(w, http.StatusOK, "series", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        series.Name,
			"ShowBar":          true,
			"ShowSearch":       false,
			"ShowAuthorSearch": false,
			"ShowSeriesSearch": false,
			"ShowViewSelector": true,
			"Title":            series.Name,
			"Books":            bl,
			"Pagination":       pagination,
		})
		return
	}

	s.render.HTML(w, http.StatusNotFound, "notfound", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Not Found",
		"ShowBar":          false,
		"ShowSearch":       false,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": false,
		"ShowViewSelector": false,
		"Title":            "Not Found",
		"Message":          "Series not found.",
	})
}

func (s *Server) handleBooks(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	userSortKey, userSortAsc := parseUserSort(r.URL.Query().Get("sort"), "id", false)

	query := storage.NewQuery().SortedBy(userSortKey, userSortAsc)
	total, err := s.storage.Books.Count(query)
	if err != nil {
		s.internalError(w, err)
		return
	}

	pagination := NewPagination(r.URL.Query(), total)
	query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit)

	bl, err := s.storage.Books.QueryDeps(query)
	if err != nil {
		s.internalError(w, err)
		return
	}

	s.render.HTML(w, http.StatusOK, "books", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Books",
		"ShowBar":          true,
		"ShowSearch":       true,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": false,
		"ShowViewSelector": true,
		"Title":            "",
		"Books":            bl,
		"Pagination":       pagination,
	})
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	bid := p.ByName("id")
	bl, err := s.storage.Books.QueryDeps(storage.NewQuery().Filtered("id", bid, true))
	if err != nil {
		s.internalError(w, err)
		return
	}

	if len(bl) > 0 {
		b := bl[0]
		s.render.HTML(w, http.StatusOK, "book", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        b.Title,
			"ShowBar":          false,
			"ShowSearch":       false,
			"ShowAuthorSearch": false,
			"ShowSeriesSearch": false,
			"ShowViewSelector": false,
			"Title":            "",
			"Book":             b,
		})
		return
	}

	s.render.HTML(w, http.StatusNotFound, "notfound", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Not Found",
		"ShowBar":          false,
		"ShowSearch":       false,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": false,
		"ShowViewSelector": false,
		"Title":            "Not Found",
		"Message":          "Book not found.",
	})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	q := r.URL.Query().Get("q")

	if len(q) != 0 {
		userSortKey, userSortAsc := parseUserSort(r.URL.Query().Get("sort"), "title", true)

		query := storage.NewQuery().SortedBy(userSortKey, userSortAsc)
		total, err := s.storage.Books.CountKeyword(q, query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		pagination := NewPagination(r.URL.Query(), total)
		bl, err := s.storage.Books.QueryKeyword(q, query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit))
		if err != nil {
			s.internalError(w, err)
			return
		}

		s.render.HTML(w, http.StatusOK, "search", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        "Search Results",
			"ShowBar":          true,
			"ShowSearch":       true,
			"ShowAuthorSearch": false,
			"ShowSeriesSearch": false,
			"ShowViewSelector": true,
			"Title":            "Search Results",
			"Query":            q,
			"Books":            bl,
			"Pagination":       pagination,
		})
		return
	}

	s.render.HTML(w, http.StatusOK, "search", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Search",
		"ShowBar":          true,
		"ShowSearch":       true,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": false,
		"ShowViewSelector": false,
		"Title":            "Search",
		"Query":            "",
	})
}

func (s *Server) handleSearchAuthors(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	q := r.URL.Query().Get("q")

	if len(q) != 0 {
		userSortKey, userSortAsc := parseUserSort(r.URL.Query().Get("sort"), "name", true)

		query := storage.NewQuery().Filtered("name", q, false).SortedBy(userSortKey, userSortAsc)
		total, err := s.storage.Authors.Count(query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		pagination := NewPagination(r.URL.Query(), total)
		query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit)

		al, err := s.storage.Authors.Query(query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		s.render.HTML(w, http.StatusOK, "authors", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        "Author Search Results",
			"ShowBar":          true,
			"ShowSearch":       false,
			"ShowAuthorSearch": true,
			"ShowSeriesSearch": false,
			"ShowViewSelector": true,
			"Title":            "Author Search Results",
			"AuthorQuery":      q,
			"Authors":          al,
			"Pagination":       pagination,
		})
		return
	}

	s.render.HTML(w, http.StatusOK, "authors", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Search Authors",
		"ShowBar":          true,
		"ShowSearch":       false,
		"ShowAuthorSearch": true,
		"ShowSeriesSearch": false,
		"ShowViewSelector": false,
		"Title":            "Search Authors",
		"AuthorQuery":      "",
	})
}

func (s *Server) handleSearchSeries(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	q := r.URL.Query().Get("q")

	if len(q) != 0 {
		userSortKey, userSortAsc := parseUserSort(r.URL.Query().Get("sort"), "name", true)

		query := storage.NewQuery().Filtered("name", q, false).SortedBy(userSortKey, userSortAsc)
		total, err := s.storage.Series.Count(query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		pagination := NewPagination(r.URL.Query(), total)
		query.Skip(pagination.ItemOffset).Take(pagination.ItemLimit)

		sl, err := s.storage.Series.Query(query)
		if err != nil {
			s.internalError(w, err)
			return
		}

		s.render.HTML(w, http.StatusOK, "seriess", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        "Series Search Results",
			"ShowBar":          true,
			"ShowSearch":       false,
			"ShowAuthorSearch": false,
			"ShowSeriesSearch": true,
			"ShowViewSelector": true,
			"Title":            "Series Search Results",
			"SeriesQuery":      q,
			"Series":           sl,
			"Pagination":       pagination,
		})
		return
	}

	s.render.HTML(w, http.StatusOK, "seriess", map[string]interface{}{
		"CurVersion":       s.version,
		"PageTitle":        "Search Series",
		"ShowBar":          true,
		"ShowSearch":       false,
		"ShowAuthorSearch": false,
		"ShowSeriesSearch": true,
		"ShowViewSelector": false,
		"Title":            "Search Series",
		"SeriesQuery":      "",
	})
}

func (s *Server) handleRandom(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	bl, err := s.storage.Books.Query(storage.NewQuery().Random().Take(1))
	if err != nil {
		s.internalError(w, err)
		return
	}

	if len(bl) == 0 {
		// empty database
		s.render.HTML(w, http.StatusOK, "search", map[string]interface{}{
			"CurVersion":       s.version,
			"PageTitle":        "Search",
			"ShowBar":          true,
			"ShowSearch":       true,
			"ShowSeriesSearch": false,
			"ShowAuthorSearch": false,
			"ShowViewSelector": false,
			"Title":            "Search",
			"Query":            "",
		})
	} else {
		b := bl[0]
		http.Redirect(w, r, fmt.Sprintf("/books/%d", b.ID), http.StatusTemporaryRedirect)
	}
}
