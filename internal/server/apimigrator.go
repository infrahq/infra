package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"

	"github.com/gin-gonic/gin"
)

type apiMigration struct {
	method          string
	path            string
	version         string
	redirect        string
	requestRewrite  func(c *gin.Context)
	responseRewrite func(c *gin.Context)
}

var migrations = []apiMigration{}

func addRedirect(a *API, method, path, newPath, version string) {
	migrations = append(migrations, apiMigration{
		method:   method,
		path:     path,
		version:  version,
		redirect: newPath,
	})
}

func addRequestRewrite[oldReq any, newReq any](a *API, method, path, version string, f func(oldReq) newReq) {
	migrations = append(migrations, apiMigration{
		method:  method,
		path:    path,
		version: version,
		requestRewrite: func(c *gin.Context) {
			reqVer := NewVersion(c.Request.Header.Get("VERSION"))
			if reqVer.GreaterThanStr(version) {
				c.Next()
				return
			}

			oldReqObj := new(oldReq)

			err := bind(c, oldReqObj)
			if err != nil {
				a.sendAPIError(c, err)
				return
			}

			newReqObj := f(*oldReqObj)

			rebuildRequest(c, newReqObj)

			c.Next()
		},
	})
}

func rebuildRequest(c *gin.Context, newReqObj interface{}) {
	query := url.Values{}
	body := map[string]interface{}{}
	r := reflect.ValueOf(newReqObj)
	t := r.Type()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if fieldName, ok := t.Field(i).Tag.Lookup("form"); ok {
			if f.Type().Name() == "uid.ID" {
				query.Add(fieldName, f.String())
				continue
			}

			// this list only needs to handle types we use with the "form" tag
			// nolint:exhaustive
			switch f.Kind() {
			case reflect.String:
				query.Add(fieldName, f.String())
			case reflect.Slice:
				// only type that does this is []uid.ID
				switch f.Elem().Type().Name() {
				case "uid.ID":
					for j := 0; j < f.Len(); j++ {
						query.Add(fieldName, f.Index(j).String())
					}
				default:
					panic("unexpected type " + f.Elem().Type().Name())
				}
			case reflect.Int, reflect.Int64:
				query.Add(fieldName, fmt.Sprintf("%d", f.Int()))
			case reflect.Uint, reflect.Uint64:
				query.Add(fieldName, fmt.Sprintf("%d", f.Int()))
			default:
				panic("unhandled reflection kind " + f.Kind().String())
			}
		}
		if fieldname, ok := t.Field(i).Tag.Lookup("json"); ok {
			body[fieldname] = f.Interface()
		}
	}
	c.Request.URL.RawQuery = query.Encode()

	if c.Request.Method != http.MethodGet {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			panic(err) // sendAPIError and return
		}
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyJSON))
	}
}

func addResponseRewrite[newResp any, oldResp any](a *API, method, path, version string, f func(newResp) oldResp) {
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
				a.sendAPIError(c, err)
				return
			}

			oldRespObj := f(*newRespObj)

			b, err := json.Marshal(oldRespObj)
			if err != nil {
				a.sendAPIError(c, err)
				return
			}

			w.body = b
			w.Flush()

			if w.flushErr != nil {
				a.sendAPIError(c, w.flushErr)
			}
		},
	})
}

func (m *apiMigration) RedirectHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.URL.Path = m.redirect
		c.Next()
	}
}

// responseWriter is made to satisfy gin.ResponseWriter, which is rather greedy with its interface demands
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
	//nolint:forcetypeassert
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotify interface.
func (w *responseWriter) CloseNotify() <-chan bool {
	//nolint:forcetypeassert
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
	//nolint:forcetypeassert
	w.ResponseWriter.(http.Flusher).Flush()
	w.flushErr = nil
}

func (w *responseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}
