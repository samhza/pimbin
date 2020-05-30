package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
)

type FilterType int

const (
	FilterDeny  FilterType = 0
	FilterAllow            = 1
)

type Server struct {
	FilterType FilterType
	Filter     []string
	UploadsDir string

	router *chi.Mux
	db     *DB
	ticker *time.Ticker
}

func NewServer(db *DB) *Server {
	t := time.NewTicker(time.Second)
	r := chi.NewRouter()
	s := &Server{
		db:     db,
		router: r,
		ticker: t,
	}
	r.Get("/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		w.Write([]byte(name))
	})
	r.Post("/", s.handleUpload)
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	form, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files := make(map[int]string)
	names := make(map[int]string)
	var index []int
	for {
		p, err := form.NextPart()
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, err.Error(),
				http.StatusInternalServerError)
			return
		}

		n := strings.Split(p.FormName(), ":")
		if len(n) != 2 {
			http.Error(w, "Invalid request", 400)
		}

		i, err := strconv.Atoi(n[1])
		if err != nil {
			http.Error(w, "Invalid request", 400)
		}

		switch n[0] {
		case "f":
			if _, ok := files[i]; ok {
				http.Error(w, "Bad request", 400)
				return
			}
			buf := bufio.NewReader(p)
			sniff, err := buf.Peek(512)
			contentType := http.DetectContentType(sniff)
			if !s.allowType(contentType) {
				http.Error(w, "Content type not allowed", 418)
				return
			}
			file, err := s.downloadFile(buf)
			if err != nil {
				http.Error(w, err.Error(),
					http.StatusInternalServerError)
				return
			}
			index = append(index, i)
			files[i] = file
			if name := p.FileName(); name != "" {
				names[i] = name
			}
		case "name":
			if _, ok := names[i]; ok {
				http.Error(w, "Bad request", 400)
				return
			}
			reader := &io.LimitedReader{R: p, N: 129}
			b := new(strings.Builder)
			_, err := io.Copy(b, reader)
			if err != nil || reader.N == 0 {
				http.Error(w, "Bad request", 400)
				return
			}
			names[i] = b.String()
		default:
			http.Error(w, "Bad request", 400)
			return
		}
	}
	sort.Ints(index)
	fmt.Println(index)
	paste := Paste{
		Files: make(map[string]string),
		Owner: "sam",
		ID:    s.id(),
	}
	for _, i := range index {
		name, ok := names[i]
		if !ok {
			name = strconv.Itoa(i)
		}
		if _, ok := paste.Files[name]; ok {
			http.Error(w, "Bad request", 400)
			return
		}
		paste.Files[name] = files[i]
		fmt.Printf("name: %v file: %v\n", name, files[i])
	}
	err = s.db.PutPaste(paste)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (s *Server) downloadFile(r io.Reader) (string, error) {
	err := os.MkdirAll(s.UploadsDir, 0750)
	if err != nil {
		return "", nil
	}
	f, err := ioutil.TempFile(s.UploadsDir, "upload-*")
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	tee := io.TeeReader(r, h)
	_, err = io.Copy(f, tee)
	if err != nil {
		return "", err
	}
	hash := base64.URLEncoding.WithPadding(
		base64.NoPadding).EncodeToString(h.Sum(nil))
	println(hash)
	err = os.Rename(f.Name(),
		filepath.Join(s.UploadsDir, hash))
	if err != nil {
		return "", err
	}
	return hash, nil
}

func (s *Server) id() string {
	<-s.ticker.C
	now := time.Now().Unix()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(now))
	return base64.URLEncoding.WithPadding(
		base64.NoPadding).EncodeToString(b)
}

func (s *Server) allowType(t string) bool {
	t = strings.Split(t, ";")[0]
	for _, f := range s.Filter {
		if f == t {
			return s.FilterType == FilterAllow
		}
	}
	return s.FilterType != FilterAllow
}
