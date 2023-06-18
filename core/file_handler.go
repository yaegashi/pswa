package core

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (c *Core) FileHandler(w http.ResponseWriter, r *http.Request) {
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
