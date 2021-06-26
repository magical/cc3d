package main

// TODO:
// - more info on level pages
// - which tiles go in which layers? what editor categories?
// - which tiles can have directions?

import (
	"compress/gzip"
	"flag"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/juju/naturalsort"
	"github.com/magical/cc3d"
	"github.com/nfnt/resize"
)

var portFlag = flag.String("port", ":8080", "port (and host) to listen for HTTP connections on")

func httpMain() {
	dirs := flag.Args()
	if len(dirs) == 0 {
		dirs = []string{"cc3d_levels"}
	}
	var mux http.ServeMux
	var h http.Handler = &mux
	tileset := loadTiles(tileSize)
	for i, levelDir := range dirs {
		if _, err := os.Stat(levelDir); err != nil {
			log.Println("warning: cannot access level dir:", err)
		}
		s := &server{
			tileset:  tileset,
			title:    "CC3D",
			levelDir: levelDir,
		}
		dirname := filepath.Base(levelDir)
		if strings.Contains(dirname, "ben10") {
			s.title = "Ben 10"
		}
		if strings.Contains(dirname, "cc3d") {
			s.externalLinks = true
		}
		go s.buildIndex()
		if i == 0 {
			mux.Handle("/", s)
		} else {
			mux.Handle("/"+dirname+"/", s)
		}
		if len(dirs) == 1 {
			h = s
		}
	}
	log.Fatal(http.ListenAndServe(*portFlag, h))
}

type server struct {
	tileset       Tileset
	levelDir      string
	title         string
	externalLinks bool
	index         sync.Map
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	_, base := path.Split(req.URL.Path)
	if base == "" {
		s.serveIndex(w, req)
	} else if strings.HasSuffix(base, "_thumb.png") {
		idStr := strings.TrimSuffix(base, "_thumb.png")
		if s.isID(idStr) {
			s.serveMap(w, req, idStr, true)
		} else {
			http.NotFound(w, req)
		}
	} else if strings.HasSuffix(base, ".png") {
		idStr := strings.TrimSuffix(base, ".png")
		if s.isID(idStr) {
			s.serveMap(w, req, idStr, false)
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

type levelInfo struct {
	Name   string
	Author string
}

func (s *server) buildIndex() {
	files, _ := filepath.Glob(filepath.Join(s.levelDir, "*.xml.gz"))
	naturalsort.Sort(files)
	for _, fullname := range files {
		fullname := fullname
		func() {
			f, err := os.Open(fullname)
			if err != nil {
				return
			}
			defer f.Close()
			m, err := cc3d.ReadLevel(f)
			if err != nil {
				return
			}
			id, _, _ := cut(filepath.Base(fullname), ".")
			s.index.Store(id, levelInfo{m.Name, m.Author})
		}()
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

var escape = template.HTMLEscapeString

func (s *server) serveIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	writeln := func(msg string, v ...interface{}) error {
		_, err := fmt.Fprintf(w, msg+"\n", v...)
		return err
	}
	writeln("<!doctype html>")
	writeln("<title>%s Level maps</title>", escape(s.title))
	writeln("<body style=\"font-family: Comic Sans MS, Chalkboard\">")
	writeln("<h1>%s Level maps</h1>", escape(s.title))
	files, _ := filepath.Glob(filepath.Join(s.levelDir, "*.xml.gz"))
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
			var err error
			if v, ok := s.index.Load(id); ok {
				li := v.(levelInfo)
				err = writeln(`<a href="%[1]s" title="%[2]s by %[3]s">%[1]s</a>`, escape(id), escape(li.Name), escape(li.Author))
			} else {
				err = writeln(`<a href="%[1]s">%[1]s</a>`, escape(id))
			}
			if err != nil {
				break
			}
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
	filename := filepath.Join(s.levelDir, id+".xml.gz")
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
	filename := filepath.Join(s.levelDir, id+".xml.gz")
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
	u := *req.URL
	u.Host = req.Host
	u.Scheme = "http" // TODO https
	pageTitle := fmt.Sprintf("%s by %s", def(m.Map.Name, "Untitled"), def(m.Map.Author, "Author Unknown"))
	writeln("<!doctype html>")
	writeln("<title>%s Levelid %s: %s</title>", escape(s.title), escape(id), escape(pageTitle))
	writeln("<meta property=\"og:title\" content=\"%s: %s\" />", escape(id), escape(pageTitle))
	writeln("<meta property=\"og:site_name\" content=\"%s levels\" />", escape(s.title))
	writeln("<meta property=\"og:type\" content=\"website\" />")
	writeln("<meta property=\"og:url\" content=\"%s\" />", escape(u.String()))
	u.Path += "_thumb.png"
	u.RawQuery = ""
	writeln("<meta property=\"og:image\" content=\"%s\" />", escape(u.String()))

	writeln("<body style=\"font-family: Comic Sans MS, Chalkboard\">")
	writeln("<h1>%s</h1>", escape(pageTitle))
	writeln("<p><img src=\"%s.png\">", escape(id))
	if !m.ModTime.IsZero() {
		writeln("<p>%s", m.ModTime.Format("Monday, January 02 2006 15:04:05 UTC"))
	}
	writeln("<p><a href=\"%s.xml\">Raw XML</a>", escape(id))
	if s.externalLinks {
		writeln("| <a rel=\"noreferrer\" href=\"https://s3.amazonaws.com/cc3d-editorreplays/hint_%s.hnt\">Replay</a>", escape(id))
		baseURL := "http://cc3d.chuckschallenge.com"
		if n, err := strconv.Atoi(id); err == nil && n < 15000 {
			baseURL = "http://beta.chuckschallenge.com"
		}
		writeln("<p><a rel=\"noreferrer\" href=\"%s/Share.php?levelId=%s\">View this level on chuckschallenge.com</a>", escape(baseURL), escape(id))
	}
}

func def(s, defaultStr string) string {
	if s == "" {
		s = defaultStr
	}
	return s
}

func (s *server) serveMap(w http.ResponseWriter, req *http.Request, id string, thumbnail bool) {
	m := s.readLevel(w, req, id)
	if m == nil {
		return
	}
	var im image.Image
	im, err := makeMap(m.Map, s.tileset, false)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}
	if thumbnail {
		im = resize.Thumbnail(200, 200, im, resize.Bilinear)
	}
	err = png.Encode(w, im)
	if err != nil {
		log.Println(err)
		// too late to change the response
		return
	}
}
