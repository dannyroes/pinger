package output

import (
	_ "embed"
	"html/template"
	"io"
)

//go:embed template.htm
var tmpl string
var t *template.Template

func GeneratePage(wr io.Writer, data any) error {
	return t.Execute(wr, data)
}

func init() {
	var err error
	t, err = template.New("results").Parse(tmpl)
	if err != nil {
		panic(err.Error())
	}
}
