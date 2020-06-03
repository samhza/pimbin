package config

import (
	"io"
	"os"

	toml "github.com/pelletier/go-toml"
)

type Server struct {
	Addr        string   `toml:"address"`
	DBPath      string   `toml:"db"`
	UploadsDir  string   `toml:"uploads"`
	BaseURL     string   `toml:"base-url"`
	FilterAllow bool     `toml:"filter-allow"`
	Filter      []string `toml:"filter-types"`
	MaxBodySize int64    `toml:"max-body-size"`
	CSSPath     string   `toml:"css"`
	SiteName    string   `toml:"name"`
}

func Defaults() *Server {
	return &Server{
		Addr:        ":3000",
		BaseURL:     "http://localhost:3000/",
		MaxBodySize: 512000000,
		DBPath:      "pimbin.db",
		UploadsDir:  "uploads",
		SiteName:    "pimbin",
	}
}

func Load(path string) (*Server, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}

func Parse(r io.Reader) (*Server, error) {
	server := Defaults()
	dec := toml.NewDecoder(r)
	if err := dec.Decode(server); err != nil {
		return nil, err
	}
	return server, nil
}
