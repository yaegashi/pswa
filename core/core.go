package core

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/auth"
	"github.com/yaegashi/pswa/config"
	"github.com/yaegashi/pswa/logging"
)

type Core struct {
	Root         string
	Config       *config.Config
	Routes       []*config.Route
	SessionStore sessions.Store
}

func (c *Core) NewMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logging.Logger(r.Context()).Sugar()

			reqPath := filepath.Clean(r.URL.Path)
			if strings.HasSuffix(r.URL.Path, "/") {
				reqPath += "/"
			}
			session, _ := c.SessionStore.Get(r, "session")
			identity, _ := session.Values["identity"].(*auth.Identity)

			var reqRoute *config.Route
			for _, rr := range c.Routes {
				if rr.Match(reqPath) {
					reqRoute = rr
					break
				}
			}

			logger.Debugf("path=%#v route=%#v", reqPath, reqRoute)

			if reqRoute == nil {
				next.ServeHTTP(w, r)
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
					next.ServeHTTP(w, r)
					return
				}
			}

			if reqRoute.Headers != nil {
				for k, v := range reqRoute.Headers {
					w.Header().Add(k, v)
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
				r.URL.Path = reqRoute.StripPrefix(r.URL.Path)
				r.URL.RawPath = r.URL.Path
				logger.Debugf("redirect to: %s", r.URL)
				reqRoute.ProxyHandler.ServeHTTP(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (c *Core) Handler(w http.ResponseWriter, r *http.Request) {
	reqPath := filepath.Clean(r.URL.Path)
	http.ServeFile(w, r, filepath.Join(c.Root, reqPath))
}

func New(root string, cfg *config.Config, ss sessions.Store) *Core {
	routes := make([]*config.Route, len(cfg.Routes))
	for i := 0; i < len(routes); i++ {
		routes[i] = cfg.Routes[i]
	}
	return &Core{
		Root:         root,
		Config:       cfg,
		SessionStore: ss,
		Routes:       routes,
	}
}
