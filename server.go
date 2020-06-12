package pimbin

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
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

	"github.com/erebid/pimbin/config"
	"github.com/go-chi/chi"
)

// Server is a pimbin server, and contains some options.
type Server struct {
	Config config.Server

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
	users, err := s.db.Users()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		s.users[u.Name] = &user{srv: s, User: u}
	}
	r.Get("/style.css", s.handleCSS)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", s.handleGetPaste)
		r.With(s.ownerCheck).Delete("/", s.handleDeletePaste)
	})
	r.Route("/raw/{hash}", func(r chi.Router) {
		r.Get("/", s.handleGetFile)
		r.Get("/{name}", s.handleGetFile)
	})
	r.With(s.ownerCheck).Post("/", s.handleUpload)
	return s, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	var username string
	if user := userFromContext(r.Context()); user != nil {
		username = user.Name
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.Config.MaxBodySize)
	paste := Paste{
		Owner: username,
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
		var i int
		switch len(n) {
		case 1:
			i = 1
		case 2:
			i, err = strconv.Atoi(n[1])
			if err != nil {
				http.Error(w, "Invalid request", 400)
			}
		}

		switch n[0] {
		case "file", "f":
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
		case "name", "n":
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
	fmt.Fprintf(w, "%s%s\n", s.Config.BaseURL, paste.ID)
}

func (s *Server) handleDeletePaste(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	if user == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	username := r.Context().Value("user").(string)
	id := chi.URLParam(r, "id")
	p, err := s.db.Paste(id)
	if err != nil || p.Owner != username {
		http.Error(w, err.Error(), 401)
		return
	}
	err = s.db.DeletePaste(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func (s *Server) handleGetPaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.db.Paste(id)
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
				s.Config.BaseURL+"raw/"+p.Files[0].Hash+"/"+p.Files[0].Name, 301)
			return
		}
		f.Close()
	}
	s.renderPaste(w, p)
}

func (s *Server) downloadFile(r io.Reader) (string, error) {
	err := os.MkdirAll(s.Config.UploadsDir, 0750)
	if err != nil {
		return "", nil
	}
	f, err := ioutil.TempFile(s.Config.UploadsDir, "upload-*")
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
		filepath.Join(s.Config.UploadsDir, hash))
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
	for _, f := range s.Config.Filter {
		if f == t {
			return s.Config.FilterAllow
		}
	}
	return !s.Config.FilterAllow
}

func (s *Server) handleCSS(w http.ResponseWriter, r *http.Request) {
	if s.Config.CSSPath != "" {
		http.ServeFile(w, r, s.Config.CSSPath)
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
	f, err := os.Open(filepath.Join(s.Config.UploadsDir, file.Hash))
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

func (s *Server) ownerCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.Config.NoAuth {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header["Authorization"]
		if len(auth) < 1 {
			http.Error(w, "no token provided", 401)
			return
		}
		token := auth[0]
		for _, u := range s.users {
			if token == u.Token {
				ctx := putUserContext(r.Context(), &u.User)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		http.Error(w, "invalid token provided", 403)
		return
	})
}

func userFromContext(ctx context.Context) *User {
	if u, ok := ctx.Value("pimbin_user").(*User); ok {
		return u
	}
	return nil
}

func putUserContext(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, "pimbin", u)
}
