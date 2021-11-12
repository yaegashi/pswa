package core

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
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

func (c *Core) Handler(w http.ResponseWriter, r *http.Request) {
	p := filepath.Join(c.Root, filepath.Clean(r.URL.Path))
	if !strings.HasSuffix(p, "/index.html") {
		http.ServeFile(w, r, p)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		msg, status := toHTTPError(err)
		http.Error(w, msg, status)
		return
	}
	defer f.Close()
	d, err := f.Stat()
	if err != nil {
		msg, status := toHTTPError(err)
		http.Error(w, msg, status)
		return
	}
	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}

func toHTTPError(err error) (msg string, httpStatus int) {
	if errors.Is(err, fs.ErrNotExist) {
		return "404 page not found", http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return "403 Forbidden", http.StatusForbidden
	}
	return "500 Internal Server Error", http.StatusInternalServerError
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
