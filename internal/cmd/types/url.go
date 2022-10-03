package types

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

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

// HostPort is used to accept hostnames or IP addresses that may include an
// optional port.
type HostPort struct {
	Host string
	Port int
}

func (h *HostPort) Set(raw string) error {
	host, port, err := net.SplitHostPort(raw)
	addrErr := &net.AddrError{}
	switch {
	case errors.As(err, &addrErr) && addrErr.Err == "missing port in address":
		h.Host = raw
		return nil
	case err != nil:
		return err
	}
	h.Host = host
	h.Port, err = strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port %q must be a number", port)
	}
	return nil
}

func (h *HostPort) String() string {
	if h.Port == 0 {
		return h.Host
	}
	return net.JoinHostPort(h.Host, strconv.Itoa(h.Port))
}

func (h *HostPort) Type() string {
	return "hostname"
}
