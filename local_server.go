package main

import (
	"context"
	"fmt"
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

func NewLocalServer() (*LocalServer, error) {
	ls := &LocalServer{ResultChan: make(chan CodeResponse, 1), srv: &http.Server{Addr: ":8301"}}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		ls.ResultChan <- CodeResponse{
			Code:  params.Get("code"),
			State: params.Get("state"),
		}
		fmt.Fprintf(w, "You may now close this window.")
	})

	go func() {
		if err := ls.srv.ListenAndServe(); err != nil {
			ls.ResultChan <- CodeResponse{Error: err}
		}
	}()

	return ls, nil
}

func (l *LocalServer) Wait() (string, string, error) {
	result := <-l.ResultChan
	l.srv.Shutdown(context.Background())
	return result.Code, result.State, result.Error
}
