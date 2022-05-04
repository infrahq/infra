package server

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

type apiMigration struct {
	method          string
	path            string
	version         string
	requestRewrite  func(c *gin.Context)
	responseRewrite func(c *gin.Context)
}

var migrations = []apiMigration{}

func addResponseRewrite[newResp any, oldResp any](method, path, version string, f func(newResp) oldResp) {
	migrations = append(migrations, apiMigration{
		method:  method,
		path:    path,
		version: version,
		responseRewrite: func(c *gin.Context) {
			reqVer := NewVersion(c.Request.Header.Get("Infra-Version"))
			if reqVer.GreaterThanStr(version) {
				c.Next()
				return
			}

			w := &responseWriter{ResponseWriter: c.Writer}
			c.Writer = w

			c.Next()

			newRespObj := new(newResp)
			err := json.Unmarshal(w.body, newRespObj)
			if err != nil {
				panic(err)
			}

			oldRespObj := f(*newRespObj)

			b, err := json.Marshal(oldRespObj)
			if err != nil {
				panic(err)
			}

			w.body = b
			w.Flush()

			if w.flushErr != nil {
				panic(w.flushErr)
			}
		},
	})
}

type responseWriter struct {
	http.ResponseWriter
	body     []byte
	size     int
	status   int
	flushErr error
}

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

var _ gin.ResponseWriter = &responseWriter{}

func (w *responseWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.size = noWritten
	w.status = defaultStatus
}

func (w *responseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		w.status = code
	}
}

func (w *responseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	w.WriteHeaderNow()
	w.body = append(w.body, data...)
	w.size += len(data)
	return len(data), nil
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	w.body = append(w.body, s...)
	w.size += len(s)
	return len(s), nil
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) Written() bool {
	return w.size != noWritten
}

// Hijack implements the http.Hijacker interface.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.size < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotify interface.
func (w *responseWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush implements the http.Flush interface.
func (w *responseWriter) Flush() {
	w.WriteHeaderNow()
	bytesToFlush := len(w.body)
	for bytesToFlush > 0 {
		bytesFlushed, err := w.ResponseWriter.Write(w.body)
		if err != nil {
			w.flushErr = err
			return
		}
		bytesToFlush -= bytesFlushed
		w.body = w.body[bytesFlushed:]
	}
	w.ResponseWriter.(http.Flusher).Flush()
	w.flushErr = nil
}

func (w *responseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
