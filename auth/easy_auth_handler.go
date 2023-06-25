package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/yaegashi/pswa/logging"
	"golang.org/x/oauth2"
)

const (
	EasyAuthPrincipalHeaderName   = "X-Ms-Client-Principal"
	EasyAuthAccessTokenHeaderName = "X-Ms-Token-Aad-Access-Token"
	EasyAuthIdTokenHeaderName     = "X-Ms-Token-Aad-Id-Token"
)

type PrincipalClaim struct {
	Typ string `json:"typ"`
	Val any    `json:"val"`
}

type Principal struct {
	AuthTyp string           `json:"auth_typ"`
	NameTyp string           `json:"name_typ"`
	RoleTyp string           `json:"role_typ"`
	Claims  []PrincipalClaim `json:"claims"`
}

func (a *Auth) EasyAuthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	if !a.EasyAuth {
		http.Error(w, "App Service Authentication not enabled", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	logger := logging.Logger(ctx).Sugar()

	xPrincipal := r.Header.Get(EasyAuthPrincipalHeaderName)
	xAccessToken := r.Header.Get(EasyAuthAccessTokenHeaderName)
	xIdToken := r.Header.Get(EasyAuthIdTokenHeaderName)

	bytePrincipal, err := base64.StdEncoding.DecodeString(xPrincipal)
	if err != nil {
		http.Error(w, fmt.Sprintf("Base64 decode failed: %s", err), http.StatusInternalServerError)
		return
	}
	var principal Principal
	err = json.Unmarshal(bytePrincipal, &principal)
	if err != nil {
		http.Error(w, fmt.Sprintf("JSON decode failed: %s", err), http.StatusInternalServerError)
	}
	principalMap := map[string]any{}
	for _, claim := range principal.Claims {
		principalMap[claim.Typ] = claim.Val
	}

	typ := "user"
	if principal.NameTyp == "appid" {
		typ = "app"
	}
	id, _ := principalMap["http://schemas.microsoft.com/identity/claims/objectidentifier"].(string)
	name, _ := principalMap["name"].(string)
	email, _ := principalMap["http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"].(string)
	groups := principalMap["groups"].([]string)

	members := make([]string, len(groups)+1)
	members[0] = strings.ToLower(id)
	for i, g := range groups {
		members[i+1] = strings.ToLower(g)
	}

	var graphGroups []string
	var graphErr error
	if groups == nil && xAccessToken != "" {
		logger.Info("No groups claim found.  Making a graph member groups request...")
		graphGroups, graphErr = GraphMemberGroupsRequest(ctx, &oauth2.Token{AccessToken: xAccessToken})
		if graphErr == nil {
			for _, g := range graphGroups {
				members = append(members, strings.ToLower(g))
			}
		} else {
			logger.Error(graphErr)
		}
	}

	identity := &Identity{
		Typ:   typ,
		Id:    id,
		Name:  name,
		Email: email,
		Roles: a.Config.MemberRoles(members),
	}
	logger.Infof("Identity: %#v", identity)

	session := a.Session(r)
	sessionReturn, _ := session.Values[ReturnValueName].(string)
	if sessionReturn == "" {
		sessionReturn = "/"
	}
	sessionDebug, _ := session.Values[DebugValueName].(string)
	delete(session.Values, StateValueName)
	delete(session.Values, ReturnValueName)
	delete(session.Values, DebugValueName)

	formReturn := r.Form.Get(ReturnValueName)
	if formReturn != "" {
		sessionReturn = formReturn
	}
	formDebug := r.Form.Get(DebugValueName)
	if formDebug != "" {
		sessionDebug = formDebug
	}

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
	fmt.Fprintf(w, `<h1>PSWA EasyAuth Handler</h1>`)
	fmt.Fprintf(w, `<p>This is the output for debugging purposes; NEVER expose any tokens to others!</p>`)
	fmt.Fprintf(w, `<p><a href="%s">Back to the application</a></p>`, sessionReturn)

	// Identity to be stored in the cookie
	fmt.Fprintf(w, `<p>Identity to be stored in the cookie:</p><pre>%s</pre>`, htmlDump(identity))

	// PSWA Configuration
	fmt.Fprintf(w, `<p>PSWA configuration:</p><pre>%s</pre>`, htmlDump(a.Config))

	// Decoded prinicipal
	fmt.Fprintf(w, `<p>Decoded principal:</p><pre>%s</pre>`, htmlDump(principal))

	// Graph member groups response
	fmt.Fprintf(w, `<p>Graph member groups response:</p>`)
	if graphErr == nil {
		fmt.Fprintf(w, `<pre>%s</pre>`, htmlDump(graphGroups))
	} else {
		fmt.Fprintf(w, `<pre>%s</pre>`, html.EscapeString(graphErr.Error()))
	}

	// EasyAuth headers
	fmt.Fprintf(w, `<p>%s:</p><pre>%s</pre>`, EasyAuthPrincipalHeaderName, html.EscapeString(xPrincipal))
	fmt.Fprintf(w, `<p>%s:</p><pre>%s</pre>`, EasyAuthAccessTokenHeaderName, html.EscapeString(xAccessToken))
	fmt.Fprintf(w, `<p>%s:</p><pre>%s</pre>`, EasyAuthIdTokenHeaderName, html.EscapeString(xIdToken))
}
