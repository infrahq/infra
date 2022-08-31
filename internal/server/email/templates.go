package email

import (
	"embed"
	htmltemplate "html/template"
	texttemplate "text/template"
)

//go:embed templates/*
var templateFiles embed.FS

var textTemplateList *texttemplate.Template
var htmlTemplateList *htmltemplate.Template

func init() {
	var err error
	textTemplateList, err = texttemplate.ParseFS(templateFiles, "**/*.text.plain")
	if err != nil {
		panic("can't read text templates: " + err.Error())
	}

	htmlTemplateList, err = htmltemplate.ParseFS(templateFiles, "**/*.text.html")
	if err != nil {
		panic("can't read html templates: " + err.Error())
	}
}
