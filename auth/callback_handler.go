package auth

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"

	"github.com/yaegashi/pswa/logging"
)

const (
	CodeValueName             = "code"
	ErrorValueName            = "error"
	ErrorDescriptionValueName = "error_description"
)

// https://learn.microsoft.com/en-us/azure/active-directory/develop/id-token-claims-reference
type ClaimNames struct {
	Groups string `json:"groups"`
}

type ClaimSources struct {
	Endpoint    string `json:"endpoint"`
	AccessToken string `json:"access_token"`
}

type Claims struct {
	Name         string                     `json:"name"`
	Email        string                     `json:"email"`
	Groups       []string                   `json:"groups"`
	ClaimNames   ClaimNames                 `json:"_claim_names"`
	ClaimSources map[string]json.RawMessage `json:"_claim_sources"`
}

func htmlDump(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return html.EscapeString(string(b))
}

func (a *Auth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if a.OAuth2Config == nil {
		http.Error(w, "OpenID Connect auth config failed: see log output", http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	logger := logging.Logger(ctx).Sugar()
	session := a.Session(r)
	sessionState, ok := session.Values[StateValueName].(string)
	if !ok {
		http.Error(w, "No state in session", http.StatusBadRequest)
		return
	}
	sessionReturn, _ := session.Values[ReturnValueName].(string)
	if sessionReturn == "" {
		sessionReturn = "/"
	}
	sessionDebug, _ := session.Values[DebugValueName].(string)
	delete(session.Values, StateValueName)
	delete(session.Values, ReturnValueName)
	delete(session.Values, DebugValueName)

	if r.FormValue(ErrorValueName) != "" {
		http.Error(w, fmt.Sprintf("Error: %s\n%s\n", r.FormValue(ErrorValueName), r.FormValue(ErrorDescriptionValueName)), http.StatusBadRequest)
		return
	}
	formCode := r.FormValue(CodeValueName)
	formState := r.FormValue(StateValueName)
	if formCode == "" || formState == "" {
		http.Error(w, "Invalid response", http.StatusBadRequest)
		return
	}
	if formState != sessionState {
		http.Error(w, "Unmatched state cookie", http.StatusBadRequest)
		return
	}
	oauth2Token, err := a.OAuth2Config.Exchange(ctx, formCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token", http.StatusBadRequest)
		return
	}
	idToken, err := a.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var claims Claims
	err = idToken.Claims(&claims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	groups := claims.Groups
	var graphGroups []string
	var graphErr error
	if groups == nil {
		logger.Info("No groups claim found.  Making a graph member groups request...")
		graphGroups, graphErr = GraphMemberGroupsRequest(ctx, oauth2Token)
		if graphErr == nil {
			groups = graphGroups
		} else {
			logger.Error(graphErr)
		}
	}

	identity := &Identity{
		Name:  claims.Name,
		Email: claims.Email,
		Roles: a.Config.MemberRoles(groups),
	}
	logger.Infof("Identity: %#v", identity)

	session.Values[IdentityValueName] = identity
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sessionDebug == "" {
		http.Redirect(w, r, sessionReturn, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=UTF-8")

	// Title and navigation
	fmt.Fprintf(w, `<style>pre { border: solid; padding: 1ex; white-space: pre-wrap; word-break: break-all; font-family: "Consolas", "Courier New", monospace; }</style>`)
	fmt.Fprintf(w, `<h1>PSWA Callback Handler</h1>`)
	fmt.Fprintf(w, `<p>This is the output for debugging purposes; NEVER expose any tokens to others!</p>`)
	fmt.Fprintf(w, `<p><a href="%s">Back to the application</a></p>`, sessionReturn)

	// Identity to be stored in the cookie
	fmt.Fprintf(w, `<p>Identity to be stored in the cookie:</p><pre>%s</pre>`, htmlDump(identity))

	// PSWA Configuration
	fmt.Fprintf(w, `<p>PSWA configuration:</p><pre>%s</pre>`, htmlDump(a.Config))

	// Decoded ID token
	fmt.Fprintf(w, `<p>Decoded ID token (name, email, groups):</p><pre>%s</pre>`, htmlDump(claims))

	// Graph member groups response
	fmt.Fprintf(w, `<p>Graph member groups response:</p>`)
	if graphErr == nil {
		fmt.Fprintf(w, `<pre>%s</pre>`, htmlDump(graphGroups))
	} else {
		fmt.Fprintf(w, `<pre>%s</pre>`, html.EscapeString(graphErr.Error()))
	}

	// Raw ID token
	fmt.Fprintf(w, `<p>Raw ID token:</p><pre>%s</pre>`, html.EscapeString(rawIDToken))

	// Raw OAuth2 token
	fmt.Fprintf(w, `<p>Raw OAuth2 tokens:</p><pre>%s</pre>`, htmlDump(oauth2Token))
}
