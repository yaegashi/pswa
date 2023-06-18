package config

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Route struct {
	Route        string            `json:"route,omitempty"`
	Rewrite      string            `json:"rewrite,omitempty"`
	Redirect     string            `json:"redirect,omitempty"`
	Proxy        string            `json:"proxy,omitempty"`
	AllowedRoles []string          `json:"allowedRoles,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	StatusCode   string            `json:"statusCode,omitempty"`
	Methods      []string          `json:"methods,omitempty"`
	ProxyHandler http.Handler      `json:"-"`
	Globber      Globber           `json:"-"`
}

func (r *Route) Compile() error {
	err := r.Globber.Compile(r.Route)
	if err != nil {
		return err
	}
	if r.Proxy != "" {
		u, err := url.Parse(r.Proxy)
		if err != nil {
			return fmt.Errorf("Route %q bad proxy URL: %w", r.Route, err)
		}
		r.ProxyHandler = httputil.NewSingleHostReverseProxy(u)
	}
	for _, ar := range r.AllowedRoles {
		if ar == "anonymous" {
			r.AllowedRoles = nil
			break
		}
		if ar == "authenticated" {
			r.AllowedRoles = []string{"authenticated"}
			break
		}
	}
	return nil
}
