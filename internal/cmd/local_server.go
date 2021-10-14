package cmd

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type ErrResultTimedOut struct{}

func (e *ErrResultTimedOut) Error() string {
	return "local server timed out waiting for result"
}

type CodeResponse struct {
	Code  string
	State string
	Error error
}

type LocalServer struct {
	ResultChan chan CodeResponse
	srv        *http.Server
}

const defaultTimeout = 300000 * time.Millisecond

//go:embed pages
var pages embed.FS

func newLocalServer() (*LocalServer, error) {
	ls := &LocalServer{ResultChan: make(chan CodeResponse, 1), srv: &http.Server{Addr: "127.0.0.1:8301"}}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		ls.ResultChan <- CodeResponse{
			Code:  params.Get("code"),
			State: params.Get("state"),
		}

		templateFile := "pages/success.html"
		templateVals := make(map[string]string)
		if params.Get("error") != "" {
			templateFile = "pages/error.html"
			templateVals["error"] = formatURLMsg(params.Get("error"), "Error")
			templateVals["error_description"] = formatURLMsg(params.Get("error_description"), "Please contact your administrator for assistance.")
		}

		t, err := template.ParseFS(pages, templateFile)
		if err != nil {
			// these templates are static so they should be able to be parsed, but this fallback ensures something is returned
			fmt.Fprintf(w, "Something went wrong, please contact your administrator for assistance.")
			return
		}

		if err := t.Execute(w, templateVals); err != nil {
			fmt.Fprintf(w, "Something went wrong, please contact your administrator for assistance.")
			return
		}
	})

	go func() {
		if err := ls.srv.ListenAndServe(); err != nil {
			ls.ResultChan <- CodeResponse{Error: err}
		}
	}()

	return ls, nil
}

func (l *LocalServer) wait(timeout time.Duration) (string, string, error) {
	var result CodeResponse

	timedOut := false

	if timeout <= 0 {
		timeout = defaultTimeout
	}

	select {
	case result = <-l.ResultChan:
		// do nothing
	case <-time.After(timeout):
		timedOut = true
	}

	if err := l.srv.Shutdown(context.Background()); err != nil {
		return "", "", err
	}

	if timedOut {
		return "", "", &ErrResultTimedOut{}
	}

	return result.Code, result.State, result.Error
}

// formatURLMsg takes a message from a URL and converts it into an easily readable form
func formatURLMsg(msg string, fallback string) string {
	if msg == "" {
		return fallback
	}

	formatted := strings.ReplaceAll(msg, "_", " ")
	formatted = strings.Title(strings.ToLower(formatted))

	return formatted
}
