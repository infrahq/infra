package email

import (
	"embed"
	htmltemplate "html/template"
	texttemplate "text/template"

	"github.com/infrahq/infra/internal/format"
)

//go:embed templates/*
var templateFiles embed.FS

var textTemplateList *texttemplate.Template
var htmlTemplateList *htmltemplate.Template

func init() {
	funcs := map[string]any{
		"humanTime": format.HumanTimeLower,
	}

	textTemplateList = texttemplate.New("text")
	textTemplateList.Funcs(funcs)
	_, err := textTemplateList.ParseFS(templateFiles, "**/*.text.plain")
	if err != nil {
		panic("can't read text templates: " + err.Error())
	}

	htmlTemplateList = htmltemplate.New("text")
	htmlTemplateList.Funcs(funcs)
	_, err = htmlTemplateList.ParseFS(templateFiles, "**/*.text.html")
	if err != nil {
		panic("can't read html templates: " + err.Error())
	}
}
