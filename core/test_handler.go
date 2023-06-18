package core

import (
	"fmt"
	"html"
	"net/http"
	"sort"
)

func (c *Core) TestHandler(w http.ResponseWriter, r *http.Request) {
	identity := c.Auth.Identity(r)

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	// Title and navigation
	fmt.Fprintf(w, `<style>pre { border: solid; padding: 1ex; white-space: pre-wrap; word-break: break-all; font-family: "Consolas", "Courier New", monospace; }</style>`)
	fmt.Fprintf(w, `<h1>PSWA Test Handler</h1>`)
	fmt.Fprintf(w, `<p>This is the output for debugging purposes; NEVER expose any tokens to others!</p>`)
	fmt.Fprintf(w, `<p><a href="/.auth/pswa/login">Login</a> / <a href="/.auth/pswa/login?debug=true">Login with debug</a> / <a href="/.auth/pswa/logout">Logout</a> / <a href="/.auth/pswa/identity">Identity</a></p>`)

	// Request Overview
	fmt.Fprintf(w, `<p>Reuqest overview:</p><pre>`)
	fmt.Fprintf(w, "<b>RemoteAddr:</b> %s\n", r.RemoteAddr)
	fmt.Fprintf(w, "<b>RequestURI:</b> %s\n", r.RequestURI)
	fmt.Fprintf(w, "<b>RewriteURI:</b> %s\n", r.URL)
	fmt.Fprintf(w, "<b>EasyAuth:</b> %v\n", c.Auth.EasyAuth)
	fmt.Fprintf(w, `</pre>`)

	// Request Headers
	fmt.Fprintf(w, `<p>Request headers:</p><pre>`)
	fmt.Fprintf(w, "<b>%s</b> %s %s\n", r.Method, r.RequestURI, r.Proto)
	fmt.Fprintf(w, "<b>Host:</b> %s\n", r.Host)
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		vals := r.Header[key]
		for _, val := range vals {
			fmt.Fprintf(w, "<b>%s:</b> %s\n", html.EscapeString(key), html.EscapeString(val))
		}
	}
	fmt.Fprintf(w, `</pre>`)

	// Identity stored in the cookie
	fmt.Fprintf(w, `<p>Identity stored in the cookie:</p><pre>%s</pre>`, htmlDump(identity))

	// PSWA Configuration
	fmt.Fprintf(w, `<p>PSWA configuration:</p><pre>%s</pre>`, htmlDump(c.Config))
}
