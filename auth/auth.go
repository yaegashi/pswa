package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/config"
	"github.com/yaegashi/pswa/logging"
	"golang.org/x/oauth2"
)

const (
	FormatAADBaseURL = "https://login.microsoftonline.com/%s/v2.0"
)

const (
	SessionCookieName         = "session"
	StateValueName            = "state"
	ReturnValueName           = "return"
	IdentityValueName         = "identity"
	DebugValueName            = "debug"
	CodeValueName             = "code"
	ErrorValueName            = "error"
	ErrorDescriptionValueName = "error_description"
)

type Auth struct {
	Provider              *oidc.Provider
	Verifier              *oidc.IDTokenVerifier
	OAuth2Config          *oauth2.Config
	OAuth2AuthCodeOptions []oauth2.AuthCodeOption
	Config                *config.Config
	SessionStore          sessions.Store
}

type Identity struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

func New(tenantID, clientID, clientSecret, redirectURI, authParams string, cfg *config.Config, ss sessions.Store) (*Auth, error) {
	baseURL := fmt.Sprintf(FormatAADBaseURL, tenantID)
	provider, err := oidc.NewProvider(context.Background(), baseURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	var authCodeOptions []oauth2.AuthCodeOption
	for _, p := range strings.Split(authParams, "&") {
		s := strings.SplitN(p, "=", 2)
		if len(s) == 2 {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam(s[0], s[1]))
		}
	}
	return &Auth{
		Provider:     provider,
		Verifier:     verifier,
		SessionStore: ss,
		Config:       cfg,
		OAuth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "User.Read"},
		},
		OAuth2AuthCodeOptions: authCodeOptions,
	}, nil
}

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

const (
	GraphMemberGroupsRequestURL  = "https://graph.microsoft.com/v1.0/me/getMemberObjects"
	GraphMemberGroupsRequestBody = `{"securityEnabledOnly":true}`
)

type GraphMemberGroupsResponse struct {
	Value []string `json:"value"`
}

func (a *Auth) GraphMemberGroupsRequest(ctx context.Context, oauth2Token *oauth2.Token) ([]string, error) {
	reqBody := bytes.NewBufferString(GraphMemberGroupsRequestBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, GraphMemberGroupsRequestURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+oauth2Token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w", err)
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %s: %s", res.Status, string(resBody))
	}
	var resGraph GraphMemberGroupsResponse
	err = json.Unmarshal(resBody, &resGraph)
	if err != nil {
		return nil, fmt.Errorf("GraphMemberGroupsRequest failed: %w: %s", err, string(resBody))
	}
	return resGraph.Value, nil
}

func (a *Auth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if a == nil {
		http.Error(w, "Auth config failed: see log output", http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	logger := logging.Logger(ctx).Sugar()
	session, _ := a.SessionStore.Get(r, SessionCookieName)
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
		graphGroups, graphErr = a.GraphMemberGroupsRequest(ctx, oauth2Token)
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
	htmldump := func(v any) string {
		b, _ := json.MarshalIndent(v, "", "  ")
		return html.EscapeString(string(b))
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<style>pre { border: solid; padding: 1ex; white-space: pre-wrap; word-break: break-all; font-family: "Consolas", "Courier New", monospace; }</style>`)
	fmt.Fprintf(w, `<p>OpenID Connect authentication debug output: NEVER expose OAuth2 tokens to others!</p>`)
	fmt.Fprintf(w, `<p><a href="%s">Back to the application</a></p>`, sessionReturn)
	fmt.Fprintf(w, `<p>Your identity:</p><pre>%s</pre>`, htmldump(identity))
	fmt.Fprintf(w, `<p>Decoded ID token (name, email, groups):</p><pre>%s</pre>`, htmldump(claims))
	fmt.Fprintf(w, `<p>Graph member groups response:</p>`)
	if graphErr == nil {
		fmt.Fprintf(w, `<pre>%s</pre>`, htmldump(graphGroups))
	} else {
		fmt.Fprintf(w, `<pre>%s</pre>`, graphErr)
	}
	fmt.Fprintf(w, `<p>Raw ID token:</p><pre>%s</pre>`, rawIDToken)
	fmt.Fprintf(w, `<p>Raw OAuth2 tokens:</p><pre>%s</pre>`, htmldump(oauth2Token))
}

func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if a == nil {
		http.Error(w, "Auth config failed: see log output", http.StatusInternalServerError)
		return
	}
	sessionState := uuid.New().String()
	sessionReturn := r.FormValue(ReturnValueName)
	if sessionReturn == "" {
		sessionReturn = r.Header.Get("Referer")
	}
	sessionDebug := r.FormValue(DebugValueName)
	session, _ := a.SessionStore.Get(r, SessionCookieName)
	session.Values[StateValueName] = sessionState
	session.Values[ReturnValueName] = sessionReturn
	session.Values[DebugValueName] = sessionDebug
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	authCodeURL := a.OAuth2Config.AuthCodeURL(sessionState, a.OAuth2AuthCodeOptions...)
	http.Redirect(w, r, authCodeURL, http.StatusFound)
}

func (a *Auth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	sessionReturn := r.FormValue(ReturnValueName)
	if sessionReturn == "" {
		sessionReturn = r.Header.Get("Referer")
	}
	if sessionReturn == "" {
		sessionReturn = "/"
	}
	if a != nil {
		session, _ := a.SessionStore.Get(r, SessionCookieName)
		session.Options.MaxAge = -1
		err := session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.SetCookie(w, &http.Cookie{Name: SessionCookieName, Path: "/", MaxAge: -1})
	}
	http.Redirect(w, r, sessionReturn, http.StatusFound)
}

func (a *Auth) MeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	body := []byte("{}")
	if a != nil {
		session, _ := a.SessionStore.Get(r, SessionCookieName)
		identity, ok := session.Values[IdentityValueName]
		if ok {
			b, err := json.Marshal(identity)
			if err == nil {
				body = b
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
