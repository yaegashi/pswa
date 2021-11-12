package config

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gobwas/glob"
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
	prefix       string
	globber      glob.Glob
}

func (r *Route) StripPrefix(path string) string {
	return strings.TrimPrefix(path, r.prefix)
}

func (r *Route) Compile() error {
	if len(r.Route) == 0 || r.Route[0] != '/' {
		return fmt.Errorf("Route %q non-absolute path", r.Route)
	}
	n := strings.LastIndexByte(r.Route, '/')
	g, err := glob.Compile(r.Route[n:])
	if err != nil {
		return fmt.Errorf("Route %q bad glob pattern: %w", r.Route, err)
	}
	r.prefix = r.Route[:n]
	r.globber = g
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

func (r *Route) Match(path string) bool {
	if !strings.HasPrefix(path, r.prefix) {
		return false
	}
	n := len(r.prefix)
	if r.Route[n:] == "/*" {
		return len(path) == n || path[n] == '/'
	}
	return r.globber.Match(path[n:])
}
