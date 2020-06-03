package pimbin

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	stdhtml "html"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

type pasteView struct {
	SiteName string
	BaseURL  string
	Paste    Paste
}

const pasteTemplate = `{{ define "paste" }}
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="{{ .BaseURL }}style.css">
  <title>{{ .SiteName }}</title>
  <meta name="description" content = "
{{ range .Paste.Files -}}
- {{ .Name }}
{{ end -}}">
</head>
<body>
{{ if lt 1 (len .Paste.Files)}}
<h1>files</h1>
<ul id="file-index">
{{ range .Paste.Files }}
<li>
<a href="#{{.Name}}">{{.Name}}</a>
</li>
{{ end }}
</ul>
{{ end }}
{{ range .Paste.Files}}
{{ if lt 1 (len $.Paste.Files)}}
{{ end }}
<h1 id="{{.Name}}" class="filename">{{.Name}}</h1>
<a href="{{ $.BaseURL }}raw/{{ .Hash }}/{{ .Name }}">raw</a>
{{ renderFile . }}
{{ end }}
</body>
</html>
{{end}}`

func (s *Server) renderPaste(w http.ResponseWriter, p *Paste) {
	funcMap := template.FuncMap{"renderFile": s.renderFile}
	t, err := template.New("paste").Funcs(funcMap).Parse(pasteTemplate)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = t.ExecuteTemplate(w, "paste", pasteView{
		BaseURL:  s.BaseURL,
		SiteName: s.SiteName,
		Paste:    *p})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (s *Server) renderFile(f File) template.HTML {
	lexer := lexers.Match(f.Name)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get("dracula")
	if style == nil {
		style = styles.Fallback
	}
	formatter := html.New(
		html.WithClasses(true),
		html.LineNumbersInTable(true),
		html.LinkableLineNumbers(true, stdhtml.EscapeString(f.Name+"-L")),
		html.WithLineNumbers(true))
	r, ctype, err := s.getPasteFile(f)
	defer r.Close()
	switch {
	case strings.HasPrefix(ctype, "text/"):
		break
	case strings.HasPrefix(ctype, "image/"):
		return template.HTML(fmt.Sprintf(`<img src="%sraw/%s" alt="%s">`,
			s.BaseURL, f.Hash, f.Name))
	default:
		return template.HTML("<p>(binary file not rendered)</p>")
	}
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return ""
	}
	iterator, err := lexer.Tokenise(nil, string(contents))
	var b strings.Builder
	err = formatter.Format(&b, style, iterator)
	if err != nil {
		return ""
	}
	return template.HTML(b.String())
}
