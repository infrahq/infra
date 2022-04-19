package types

import (
	"net/url"

	"github.com/goware/urlx"
)

// URL is an alias for url.URL that allows it to be parsed from a command line
// flag, or config file.
type URL url.URL

func (u *URL) Set(raw string) error {
	v, err := urlx.Parse(raw)
	if err != nil {
		return err
	}
	*u = URL(*v)
	return nil
}

func (u *URL) String() string {
	if u == nil {
		return ""
	}
	return (*url.URL)(u).String()
}

func (u *URL) Type() string {
	return "url"
}

func (u *URL) Value() *url.URL {
	return (*url.URL)(u)
}
