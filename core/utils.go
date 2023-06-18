package core

import (
	"encoding/json"
	"errors"
	"html"
	"io/fs"
	"net/http"
)

func toHTTPError(err error) (msg string, httpStatus int) {
	if errors.Is(err, fs.ErrNotExist) {
		return "404 page not found", http.StatusNotFound
	}
	if errors.Is(err, fs.ErrPermission) {
		return "403 Forbidden", http.StatusForbidden
	}
	return "500 Internal Server Error", http.StatusInternalServerError
}

func htmlDump(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return html.EscapeString(string(b))
}
