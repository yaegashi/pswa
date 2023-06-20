package core

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func fileErrorStatus(err error) (httpStatus int) {
	if errors.Is(err, fs.ErrNotExist) {
		return http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

func (c *Core) FileHandler(w http.ResponseWriter, r *http.Request) {
	p := filepath.Join(c.Root, filepath.Clean(r.URL.Path))
	if !strings.HasSuffix(p, "/index.html") {
		http.ServeFile(w, r, p)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		httpWriteError(w, r, fileErrorStatus(err), "")
		return
	}
	defer f.Close()
	d, err := f.Stat()
	if err != nil {
		httpWriteError(w, r, fileErrorStatus(err), "")
		return
	}
	http.ServeContent(w, r, d.Name(), d.ModTime(), f)
}
