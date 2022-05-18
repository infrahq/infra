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
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"

	"github.com/infrahq/infra/internal"
	"github.com/infrahq/infra/uid"
)

type apiMigration struct {
	method          string
	path            string
	version         string
	redirect        string
	requestRewrite  func(c *gin.Context)
	responseRewrite func(c *gin.Context)
	redirectHandler func(c *gin.Context)
	index           int
}

func addRedirect(a *API, method, path, newPath, version string, optMiddleware ...gin.HandlerFunc) {
	var optRedirectMiddleware gin.HandlerFunc
	if len(optMiddleware) > 0 {
		optRedirectMiddleware = optMiddleware[0]
	}
	a.migrations = append(a.migrations, apiMigration{
		method:          method,
		path:            path,
		version:         version,
		redirect:        newPath,
		redirectHandler: optRedirectMiddleware,
		index:           len(a.migrations),
	})
}

func addRequestRewrite[oldReq any, newReq any](a *API, method, path, version string, f func(oldReq) newReq) {
	migrationVersion, err := semver.NewVersion(version)
	if err != nil {
		panic(err) // dev mistake
	}
	a.migrations = append(a.migrations, apiMigration{
		method:  method,
		path:    path,
		version: version,
		index:   len(a.migrations),
		requestRewrite: func(c *gin.Context) {
			if !rewriteRequired(c, migrationVersion) {
				c.Next()
				return
			}

			oldReqObj := new(oldReq)

			err = bind(c, oldReqObj)
			if err != nil {
				sendAPIError(c, err)
				return
			}

			newReqObj := f(*oldReqObj)

			rebuildRequest(c, newReqObj)

			c.Next()
		},
	})
}

func rewriteRequired(c *gin.Context, migrationVersion *semver.Version) bool {
	headerVer := c.Request.Header.Get("Infra-Version")
	if headerVer == "" {
		// remove this conditional in v0.15.0
		headerVer = "0.0.0"
	}
	if headerVer == "" {
		sendAPIError(c, fmt.Errorf("%w: Infra-Version header required", internal.ErrBadRequest))
		return false
	}
	reqVer, err := semver.NewVersion(headerVer)
	if err != nil {
		sendAPIError(c, fmt.Errorf("%w: invalid Infra-Version header: %s", internal.ErrBadRequest, err))
		return false
	}

	return reqVer.LessThan(migrationVersion) || reqVer.Equal(migrationVersion)
}

func rebuildRequest(c *gin.Context, newReqObj interface{}) {
	query := url.Values{}
	body := map[string]interface{}{}
	r := reflect.ValueOf(newReqObj)
	t := r.Type()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if fieldName, ok := t.Field(i).Tag.Lookup("form"); ok {
			if structNameWithPkg(f.Type()) == "uid.ID" {
				query.Add(fieldName, uid.ID(f.Int()).String())
				continue
			}

			// this list only needs to handle types we use with the "form" tag
			// nolint:exhaustive
			switch f.Kind() {
			case reflect.String:
				query.Add(fieldName, f.String())
			case reflect.Slice:
				// only type that does this is []uid.ID
				switch structNameWithPkg(f.Type()) {
				case "uid.ID":
					for j := 0; j < f.Len(); j++ {
						query.Add(fieldName, uid.ID(f.Index(j).Int()).String())
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
			fieldname = strings.SplitN(fieldname, ",", 2)[0]
			body[fieldname] = f.Interface()
		}
	}
	c.Request.URL.RawQuery = query.Encode()

	switch c.Request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			sendAPIError(c, err)
			return
		}
		c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyJSON))
	}
}

func addResponseRewrite[newResp any, oldResp any](a *API, method, path, version string, f func(newResp) oldResp) {
	migrationVersion, err := semver.NewVersion(version)
	if err != nil {
		panic(err) // dev mistake
	}

	a.migrations = append(a.migrations, apiMigration{
		method:  method,
		path:    path,
		version: version,
		index:   len(a.migrations),
		responseRewrite: func(c *gin.Context) {
			if !rewriteRequired(c, migrationVersion) {
				c.Next()
				return
			}

			w := &responseWriter{ResponseWriter: c.Writer}
			c.Writer = w

			c.Next()

			if w.status < 300 && len(w.body) > 0 {
				newRespObj := new(newResp)
				err := json.Unmarshal(w.body, newRespObj)
				if err != nil {
					sendAPIError(c, err)
					return
				}

				oldRespObj := f(*newRespObj)

				b, err := json.Marshal(oldRespObj)
				if err != nil {
					sendAPIError(c, err)
					return
				}

				w.body = b
			}
			w.Flush()

			if w.flushErr != nil {
				sendAPIError(c, w.flushErr)
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

type HTTPMethodBindFunc func(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes

func bindRoute(a *API, method, path string, handler gin.HandlerFunc, methodFunc HTTPMethodBindFunc) {
	// build up the handlers into a map of all the paths we need to bind into.
	routes := map[string][]gin.HandlerFunc{}
	// set the default path
	routes[path] = []gin.HandlerFunc{handler}

	// we're going to build this list in referse order, prepending middleware.
	// we start with the current migration and prepend versions backwards,
	// 0.1.3, then 0.1.2, then 0.1.1.
	sort.Slice(a.migrations, sortVersionDescendingOrder(a.migrations))
	for _, migration := range a.migrations {
		if strings.ToUpper(migration.method) != method {
			continue
		}
		route, ok := routes[migration.path]
		// nolint:gocritic
		if ok {
			if migration.requestRewrite != nil {
				route = append([]gin.HandlerFunc{migration.requestRewrite}, route...)
			}
			if migration.responseRewrite != nil {
				route = append([]gin.HandlerFunc{migration.responseRewrite}, route...)
			}
		} else if len(migration.redirect) > 0 {
			// Redirects end up duplicating/splitting into a new path without destroying the old one
			route, ok = routes[migration.redirect]
			if !ok {
				panic(fmt.Sprintf("invalid migration: there is no http %s route named %q defined to redirect to", migration.method, migration.redirect))
			}
			route = append([]gin.HandlerFunc{migration.RedirectHandler()}, route...)
			if migration.redirectHandler != nil {
				// if the migration has a custom redirect handler, prepend it.
				route = append([]gin.HandlerFunc{migration.redirectHandler}, route...)
			}
		} else {
			panic(fmt.Sprintf("invalid migration: there is no http %s route named %q to rewrite", migration.method, migration.path))
		}
		routes[migration.path] = route
	}

	// now bind all relevant paths with Gin
	for path, handlers := range routes {
		methodFunc(path, handlers...)
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
	noWritten = -1
)

var _ gin.ResponseWriter = &responseWriter{}

func (w *responseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		w.status = code
	}
}

func (w *responseWriter) WriteHeaderNow() {
	w.ResponseWriter.WriteHeader(w.status)
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.body = append(w.body, s...)
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
	type closeNotifier interface {
		CloseNotify() <-chan bool
	}
	if cn, ok := w.ResponseWriter.(closeNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

// Flush implements the http.Flush interface.
func (w *responseWriter) Flush() {
	w.WriteHeaderNow()
	bytesToFlush := len(w.body)
	w.size = bytesToFlush
	for bytesToFlush > 0 {
		bytesFlushed, err := w.ResponseWriter.Write(w.body[w.size-bytesToFlush:])
		if err != nil {
			w.flushErr = err
			return
		}
		bytesToFlush -= bytesFlushed
	}
	w.flushErr = nil
}

func (w *responseWriter) Pusher() (pusher http.Pusher) {
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}

func sortVersionDescendingOrder(m []apiMigration) func(i, j int) bool {
	return func(i, j int) bool {
		iver, _ := semver.NewVersion(m[i].version)
		jver, _ := semver.NewVersion(m[j].version)
		if iver.LessThan(jver) {
			return false
		}
		if iver.GreaterThan(jver) {
			return true
		}
		return m[i].index > m[j].index
	}
}
