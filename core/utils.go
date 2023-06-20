package core

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
)

func htmlDump(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return html.EscapeString(string(b))
}

func httpWriteError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	fmt.Fprintf(w, `<h1>%d %s</h1>`, status, http.StatusText(status))
	if msg != "" {
		fmt.Fprintf(w, `<p>%s</p>`, html.EscapeString(msg))
	}
	if status == http.StatusForbidden {
		fmt.Fprintf(w, `<p><a href="/.auth/pswa/login">Sign in with another account</a></p>`)
	}
}
