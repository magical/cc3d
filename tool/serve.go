package main

// TODO:
// - more info on level pages
// - which tiles go in which layers? what editor categories?
// - which tiles can have directions?
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

	"github.com/juju/naturalsort"
	"github.com/magical/cc3d"
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
		if s.isID(idStr) {
			s.serveMap(w, req, idStr)
		} else {
			http.NotFound(w, req)
		}
	} else if strings.HasSuffix(base, ".xml") {
		idStr := strings.TrimSuffix(base, ".xml")
		if s.isID(idStr) {
			s.serveXML(w, req, idStr)
		} else {
			http.NotFound(w, req)
		}
	} else if !strings.Contains(base, ".") {
		if s.isID(base) {
			s.serveInfo(w, req, base)
		} else {
			http.NotFound(w, req)
		}
	} else {
		http.NotFound(w, req)
	}
}

// Reports whether idStr looks like a valid levelid.
// Might not actually be valid.
func (s *server) isID(idStr string) bool {
	if _, err := strconv.ParseInt(idStr, 10, 64); err == nil {
		return true
	}
	return false
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
	writeln("<body style=\"font-family: Comic Sans MS, Chalkboard\">")
	writeln("<h1>CC3d Level maps</h1>")
	files, _ := filepath.Glob(filepath.Join("cc3d_levels", "*.xml.gz"))
	naturalsort.Sort(files)
	writeln("<p>")
	for _, fullname := range files {
		name := filepath.Base(fullname)
		id, _, _ := cut(name, ".")
		if n, err := strconv.Atoi(id); err == nil {
			if n == 1004 || n == 16501 || n == 17001 || n%250 == 0 {
				writeln("<p>")
			}
		}
		if s.isID(id) {
			writeln(`<a href="%[1]s">%[1]s</a>`, escape(id))
		}
	}
}

type Map struct {
	*cc3d.Map
	ModTime time.Time
}

// Read the level with the given id.
// Returns nil and prints an error if the level isn't found an error occurs during parsing.
func (s *server) readLevel(w http.ResponseWriter, req *http.Request, id string) *Map {
	filename := filepath.Join("cc3d_levels", id+".xml.gz")
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
	mtime := zr.Header.ModTime.UTC()
	m, err := cc3d.ReadLevel(zr)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return nil
	}
	return &Map{m, mtime}
}

func (s *server) serveXML(w http.ResponseWriter, req *http.Request, id string) {
	filename := filepath.Join("cc3d_levels", id+".xml.gz")
	f, err := os.Open(filename)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	var r io.Reader
	if acceptsGzip(req) {
		w.Header().Set("Content-Encoding", "gzip")
		r = f
	} else {
		zr, err := gzip.NewReader(f)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), 500)
			return
		}
		r = zr
	}
	w.Header().Set("Content-Type", "application/xml")
	_, err = io.Copy(w, r)
	if err != nil {
		log.Println(err)
	}
}

func acceptsGzip(req *http.Request) bool {
	for _, s := range strings.Split(req.Header.Get("Accept-Encoding"), ",") {
		v, params, _ := cut(strings.TrimSpace(s), ";")
		_ = params
		if v == "gzip" {
			return true
		}
	}
	return false
}

func (s *server) serveInfo(w http.ResponseWriter, req *http.Request, id string) {
	m := s.readLevel(w, req, id)
	if m == nil {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeln := func(msg string, v ...interface{}) {
		fmt.Fprintf(w, msg+"\n", v...)
	}
	writeln("<!doctype html>")
	writeln("<title>CC3d Levelid %s: %s by %s</title>", escape(id), escape(m.Map.Name), escape(m.Map.Author))
	writeln("<body style=\"font-family: Comic Sans MS, Chalkboard\">")
	writeln("<h1>%s by %s</h1>", escape(def(m.Map.Name, "Untitled")), escape(def(m.Map.Author, "Author Unknown")))
	writeln("<p><img src=\"%s.png\">", escape(id))
	if !m.ModTime.IsZero() {
		writeln("<p>%s", m.ModTime.Format("Monday, January 02 2006 15:04:05 UTC"))
	}
	writeln("<p><a href=\"%s.xml\">Raw XML</a>", escape(id))
	writeln("| <a rel=\"noreferrer\" href=\"https://s3.amazonaws.com/cc3d-editorreplays/hint_%s.hnt\">Replay</a>", escape(id))
	baseURL := "http://cc3d.chuckschallenge.com"
	if n, err := strconv.Atoi(id); err == nil && n < 15000 {
		baseURL = "http://beta.chuckschallenge.com"
	}
	writeln("<p><a rel=\"noreferrer\" href=\"%s/Share.php?levelId=%s\">View this level on chuckschallenge.com</a>", escape(baseURL), escape(id))
}

func def(s, defaultStr string) string {
	if s == "" {
		s = defaultStr
	}
	return s
}

func (s *server) serveMap(w http.ResponseWriter, req *http.Request, id string) {
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
