package core

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yaegashi/pswa/config"
	"github.com/yaegashi/pswa/logging"
)

func (c *Core) NewMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logging.Logger(r.Context()).Sugar()

			identity := c.Auth.Identity(r)

			reqPath := filepath.Clean(r.URL.Path)
			if strings.HasSuffix(r.URL.Path, "/") && !strings.HasPrefix(reqPath, "/") {
				reqPath += "/"
			}

			var reqRoute *config.Route
			for _, rr := range c.Routes {
				if rr.Globber.Match(reqPath) {
					reqRoute = rr
					break
				}
			}

			//logger.Debugf("path=%#v route=%#v", reqPath, reqRoute)

			fallback := func() {
				if c.Config.NavigationFallback != nil {
					ok := false
					for _, g := range c.Config.NavigationFallback.Globbers {
						if g.Match(reqPath) {
							ok = true
							break
						}
					}
					if !ok {
						r = r.Clone(r.Context())
						r.URL.Path = c.Config.NavigationFallback.Rewrite
						r.URL.RawPath = c.Config.NavigationFallback.Rewrite
						w.Header().Set("Cache-Control", "no-cache")
					}
				}
				next.ServeHTTP(w, r)
			}

			if reqRoute == nil {
				fallback()
				return
			}

			if reqRoute.Methods != nil {
				ok := false
				method := strings.ToLower(r.Method)
				for _, m := range reqRoute.Methods {
					if strings.ToLower(m) == method {
						ok = true
						break
					}
				}
				if !ok {
					fallback()
					return
				}
			}

			if reqRoute.AllowedRoles != nil {
				w.Header().Set("Cache-Control", "no-cache")
			}

			if reqRoute.Headers != nil {
				for k, v := range reqRoute.Headers {
					w.Header().Set(k, v)
				}
			}

			if reqRoute.AllowedRoles != nil {
				if identity == nil {
					redirectURL := fmt.Sprintf("/.auth/login/aad?return=%s", url.QueryEscape(r.URL.String()))
					http.Redirect(w, r, redirectURL, http.StatusFound)
					return
				}
				ok := false
				for _, role := range reqRoute.AllowedRoles {
					n := sort.Search(len(identity.Roles), func(i int) bool { return identity.Roles[i] >= role })
					if n < len(identity.Roles) && identity.Roles[n] == role {
						ok = true
						break
					}
				}
				if !ok {
					http.Error(w, "403 Forbidden", http.StatusForbidden)
					return
				}
			}

			if reqRoute.Redirect != "" {
				http.Redirect(w, r, reqRoute.Redirect, http.StatusFound)
				return
			}

			if reqRoute.Rewrite != "" {
				r = r.Clone(r.Context())
				r.URL.Path = reqRoute.Rewrite
				r.URL.RawPath = reqRoute.Rewrite
				next.ServeHTTP(w, r)
				return
			}

			if reqRoute.ProxyHandler != nil {
				r = r.Clone(r.Context())
				r.URL.Path = reqRoute.Globber.StripPrefix(r.URL.Path)
				r.URL.RawPath = r.URL.Path
				logger.Debugf("redirect to: %s", r.URL)
				reqRoute.ProxyHandler.ServeHTTP(w, r)
				return
			}

			fallback()
		})
	}
}
