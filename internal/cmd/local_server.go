package cmd

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
)

type CodeResponse struct {
	Code  string
	State string
	Error error
}

type LocalServer struct {
	ResultChan chan CodeResponse
	srv        *http.Server
}

func newLocalServer() (*LocalServer, error) {
	ls := &LocalServer{ResultChan: make(chan CodeResponse, 1), srv: &http.Server{Addr: "127.0.0.1:8301"}}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		ls.ResultChan <- CodeResponse{
			Code:  params.Get("code"),
			State: params.Get("state"),
		}
		t, err := template.ParseFiles("./internal/cmd/pages/success.html")
		if err != nil {
			// the template is static so it should be able to be parsed, but this fallback ensures something is returned
			fmt.Fprintf(w, "You may now close this window.")
			return
		}
		t.Execute(w, nil)
	})

	go func() {
		if err := ls.srv.ListenAndServe(); err != nil {
			ls.ResultChan <- CodeResponse{Error: err}
		}
	}()

	return ls, nil
}

func (l *LocalServer) wait() (string, string, error) {
	result := <-l.ResultChan
	l.srv.Shutdown(context.Background())
	return result.Code, result.State, result.Error
}
