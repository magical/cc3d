package main

// TODO:
// - more info on level pages
// - which tiles go in which layers? what editor categories?
// - make rotated reflector tiles
// - make open toggle doors

import (
	"compress/gzip"
	"flag"
	"fmt"
	"html/template"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/magical/cc3d"

	"github.com/juju/naturalsort"
)

var portFlag = flag.String("port", ":8080", "port (and host) to listen for HTTP connections on")

func httpMain() {
	tileset := loadTiles(tileSize)
	s := &server{tileset}
	log.Fatal(http.ListenAndServe(*portFlag, s))
}

type server struct {
	tileset Tileset
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	} else if req.URL.Path == "/" {
		s.serveIndex(w, req)
	} else if base := path.Base(req.URL.Path); strings.HasSuffix(base, ".png") {
		idStr := strings.TrimSuffix(base, ".png")
		if id, err := strconv.ParseInt(idStr, 10, 64); err == nil {
			s.serveMap(w, req, id)
		} else {
			http.NotFound(w, req)
		}

	} else if !strings.Contains(base, ".") {
		if id, err := strconv.ParseInt(base, 10, 64); err == nil {
			s.serveInfo(w, req, id)
		} else {
			http.NotFound(w, req)
		}
	} else {
		http.NotFound(w, req)
	}
}

const levelDir = "cc3d_levels"

var escape = template.HTMLEscapeString

func (s *server) serveIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeln := func(msg string, v ...interface{}) {
		fmt.Fprintf(w, msg+"\n", v...)
	}
	writeln("<!doctype html>")
	writeln("<title>CC3d Level maps</title>")
	writeln("<body style=\"font-family: Comic Sans MS\">")
	writeln("<h1>CC3d Level maps</h1>")
	files, _ := filepath.Glob(filepath.Join("cc3d_levels", "*.xml.gz"))
	naturalsort.Sort(files)
	for i, fullname := range files {
		if i%250 == 0 {
			writeln("<p>")
		}
		name := filepath.Base(fullname)
		id, _, _ := cut(name, ".")
		if _, err := strconv.ParseInt(id, 10, 0); err == nil {
			writeln(`<a href="%[1]s">%[1]s</a>`, escape(id))
		}
	}
}

type Map struct {
	*cc3d.Map
	ModTime time.Time
}

type namedReader struct {
	io.Reader
	name string
}

func (r namedReader) Name() string {
	return r.name
}

// Read the level with the given id.
// Returns nil and prints an error if the level isn't found an error occurs during parsing.
func (s *server) readLevel(w http.ResponseWriter, req *http.Request, id int64) *Map {
	filename := filepath.Join("cc3d_levels", strconv.Itoa(int(id))+".xml.gz")
	f, err := os.Open(filename)
	if err != nil {
		http.NotFound(w, req)
		return nil
	}
	zr, err := gzip.NewReader(f)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return nil
	}
	mtime := zr.Header.ModTime
	m, err := cc3d.ReadLevel(namedReader{zr, zr.Header.Name})
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return nil
	}
	return &Map{m, mtime}
}

func (s *server) serveInfo(w http.ResponseWriter, req *http.Request, id int64) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeln := func(msg string, v ...interface{}) {
		fmt.Fprintf(w, msg+"\n", v...)
	}
	writeln("<!doctype html>")
	writeln("<title>CC3d Levelid %d</title>", id)
	writeln("<body style=\"font-family: Comic Sans MS\">")
	writeln("<p><img src=\"%d.png\">", id)
	baseURL := "http://cc3d.chuckschallenge.com"
	if id < 15000 {
		baseURL = "http://beta.chuckschallenge.com"
	}
	writeln("<p><a href=\"%s/Share.php?levelId=%d\">View this level on chuckschallenge.com</a>", escape(baseURL), id)
}

func (s *server) serveMap(w http.ResponseWriter, req *http.Request, id int64) {
	m := s.readLevel(w, req, id)
	if m == nil {
		return
	}
	im, err := makeMap(m.Map, s.tileset, false)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	err = png.Encode(w, im)
	if err != nil {
		log.Println(err)
		// too late to change the response
		return
	}
}
