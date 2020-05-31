package main

type pasteView struct {
	BaseURL string
	Paste   Paste
}

const pasteTemplate = `{{ define "paste" }}
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="{{ .BaseURL }}style.css">
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
<a href="#{{.Name}}">#</a>
<h1 id="{{.Name}}" class="filename">{{.Name}}</h1>
<a href="{{ $.BaseURL }}blob/{{ .Hash }}/{{ .Name }}">raw</a>
{{ renderChroma . }}
{{ end }}
</body>
</html>
{{end}}`
