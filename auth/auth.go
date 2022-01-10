package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/config"
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

func dump(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

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
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		OAuth2AuthCodeOptions: authCodeOptions,
	}, nil
}

func (a *Auth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if a == nil {
		http.Error(w, "Auth config failed: see log output", http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
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
	oauth2Token, err := a.OAuth2Config.Exchange(context.Background(), formCode)
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
	var claims struct {
		Name   string   `json:"name"`
		Email  string   `json:"email"`
		Groups []string `json:"groups"`
	}
	err = idToken.Claims(&claims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	identity := &Identity{
		Name:  claims.Name,
		Email: claims.Email,
		Roles: a.Config.MemberRoles(claims.Groups),
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
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "OpenID Connect authentication debug output: NEVER expose OAuth2 tokens to others!\n\n")
	fmt.Fprint(w, "Your identity:\n\n")
	dump(w, identity)
	fmt.Fprint(w, "\nDecoded ID token (name, email, groups):\n\n")
	dump(w, claims)
	fmt.Fprint(w, "\nRaw ID token:\n\n")
	dump(w, rawIDToken)
	fmt.Fprint(w, "\nOAuth2 tokens:\n\n")
	dump(w, oauth2Token)
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
