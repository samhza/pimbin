package pimbin

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
)

// Server is a pimbin server, and contains some options.
type Server struct {
	FilterAllow bool
	Filter      []string
	UploadsDir  string
	CSSPath     string
	BaseURL     string
	SiteName    string
	MaxBodySize int64

	router *chi.Mux
	db     *DB
	ticker *time.Ticker
	users  map[string]*user
}

// NewServer returns a new Server that uses the given database.
func NewServer(db *DB) (*Server, error) {
	t := time.NewTicker(time.Second)
	r := chi.NewRouter()
	s := &Server{
		db:     db,
		router: r,
		ticker: t,
		users:  make(map[string]*user),
	}
	users, err := s.db.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		s.users[u.Name] = &user{srv: s, User: u}
	}
	r.Get("/style.css", s.handleCSS)
	r.Get("/{id}", s.handleGetPaste)
	r.Delete("/{id}", s.handleDeletePaste)
	r.Get("/raw/{hash}", s.handleGetFile)
	r.Get("/raw/{hash}/{name}", s.handleGetFile)
	r.Post("/", s.handleUpload)
	return s, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	user, err := s.authorize(r)
	if err != nil {
		http.Error(w, err.Error(), 403)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.MaxBodySize)
	paste := Paste{
		Owner: user.Name,
	}
	form, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files := make(map[int]string)
	names := make(map[int]string)
	types := make(map[int]string)
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
			types[i] = contentType
			if name := p.FileName(); name != "" {
				names[i] = name
			}
		case "name":
			reader := &io.LimitedReader{R: p, N: 129}
			b := new(strings.Builder)
			_, err := io.Copy(b, reader)
			if err != nil || reader.N == 0 {
				http.Error(w, "Bad request", 400)
				return
			}
			name := b.String()
			for _, n := range names {
				if n == name {
					http.Error(w, "Bad request", 400)
					return
				}
			}
			names[i] = name
		default:
			http.Error(w, "Bad request", 400)
			return
		}
	}
	sort.Ints(index)
	paste.ID = s.id()
	for _, i := range index {
		name, ok := names[i]
		if !ok {
			if len(index) == 1 {
				name = ""
			} else {
				name = strconv.Itoa(i)
			}
			if exts, err := mime.ExtensionsByType(types[i]); err == nil {
				name = name + exts[0]
			}
		}
		file := File{
			Hash: files[i],
			Name: name,
		}
		paste.Files = append(paste.Files, file)
	}
	err = s.db.PutPaste(paste)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	fmt.Fprintf(w, "%s%s\n", s.BaseURL, paste.ID)
}

func (s *Server) handleDeletePaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.db.GetPaste(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	user, err := s.authorize(r)
	if err != nil || p.Owner != user.Name {
		http.Error(w, err.Error(), 403)
		return
	}
	err := s.db.DeletePaste(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) handleGetPaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.db.GetPaste(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(p.Files) == 1 {
		f, ctype, err := s.getPasteFile(p.Files[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if ctype != "text/plain" {
			http.Redirect(w, r,
				s.BaseURL+"raw/"+p.Files[0].Hash+"/"+p.Files[0].Name, 301)
			return
		}
		f.Close()
	}
	s.renderPaste(w, p)
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
			return s.FilterAllow
		}
	}
	return !s.FilterAllow
}

func (s *Server) handleCSS(w http.ResponseWriter, r *http.Request) {
	if s.CSSPath != "" {
		http.ServeFile(w, r, s.CSSPath)
		return
	}
	reader := strings.NewReader(defaultCSS)
	http.ServeContent(w, r, "style.css", time.Time{}, reader)
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	name := chi.URLParam(r, "name")
	f, ctype, err := s.getPasteFile(File{Hash: hash, Name: name})
	defer f.Close()
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	w.Header().Set("Content-Type", ctype)
	http.ServeContent(w, r, name, time.Time{}, f)
}

func (s *Server) getPasteFile(file File) (*os.File, string, error) {
	f, err := os.Open(filepath.Join(s.UploadsDir, file.Hash))
	if err != nil {
		return nil, "", err
	}
	ctype := mime.TypeByExtension(filepath.Ext(file.Name))
	if ctype == "" {
		var buf [512]byte
		n, _ := io.ReadFull(f, buf[:])
		ctype = http.DetectContentType(buf[:n])
		_, err := f.Seek(0, io.SeekStart)
		if err != nil {
			return nil, "", err
		}
	}
	if strings.HasPrefix(ctype, "text/") {
		ctype = "text/plain"
	}
	return f, ctype, nil
}

func (s *Server) authorize(r *http.Request) (*user, error) {
	auth := r.Header["Authorization"]
	if len(auth) < 1 {
		return nil, errors.New("no token provided")
	}
	token := auth[0]
	for _, u := range s.users {
		if token == u.Token {
			return u, nil
		}
	}
	return nil, errors.New(("invalid token provided"))
}
