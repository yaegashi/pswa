package config

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Route struct {
	Route        string            `json:"route"`
	Rewrite      string            `json:"rewrite"`
	Redirect     string            `json:"redirect"`
	Proxy        string            `json:"proxy"`
	AllowedRoles []string          `json:"allowedRoles"`
	Headers      map[string]string `json:"headers"`
	StatusCode   string            `json:"statusCode"`
	Methods      []string          `json:"methods"`
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
