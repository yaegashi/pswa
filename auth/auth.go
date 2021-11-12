package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/config"
	"golang.org/x/oauth2"
)

const (
	FormatAADBaseURL = "https://login.microsoftonline.com/%s/v2.0"
)

func dump(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

type Auth struct {
	Provider     *oidc.Provider
	Verifier     *oidc.IDTokenVerifier
	OAuth2Config *oauth2.Config
	Config       *config.Config
	SessionStore sessions.Store
}

type Identity struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

func New(tenantID, clientID, clientSecret, redirectURL string, cfg *config.Config, ss sessions.Store) (*Auth, error) {
	baseURL := fmt.Sprintf(FormatAADBaseURL, tenantID)
	provider, err := oidc.NewProvider(context.Background(), baseURL)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	return &Auth{
		Provider:     provider,
		Verifier:     verifier,
		SessionStore: ss,
		Config:       cfg,
		OAuth2Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}, nil
}

func (a *Auth) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session, _ := a.SessionStore.Get(r, "session")
	sessionState, ok := session.Values["state"].(string)
	if !ok {
		http.Error(w, "No state in session", http.StatusBadRequest)
		return
	}
	sessionReturn, _ := session.Values["return"].(string)
	if sessionReturn == "" {
		sessionReturn = "/"
	}
	sessionDebug, _ := session.Values["debug"].(string)
	delete(session.Values, "state")
	delete(session.Values, "return")
	delete(session.Values, "debug")

	if r.FormValue("error") != "" {
		http.Error(w, fmt.Sprintf("Error: %s\n%s\n", r.FormValue("error"), r.FormValue("error_description")), http.StatusBadRequest)
		return
	}
	formCode := r.FormValue("code")
	formState := r.FormValue("state")
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
	session.Values["identity"] = identity
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
	dump(w, oauth2Token)
	dump(w, rawIDToken)
	dump(w, claims)
	dump(w, identity)
}

func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	sessionState := uuid.New().String()
	sessionReturn := r.FormValue("return")
	if sessionReturn == "" {
		sessionReturn = r.Header.Get("Referer")
	}
	sessionDebug := r.FormValue("debug")
	session, _ := a.SessionStore.Get(r, "session")
	session.Values["state"] = sessionState
	session.Values["return"] = sessionReturn
	session.Values["debug"] = sessionDebug
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	authCodeURL := a.OAuth2Config.AuthCodeURL(sessionState)
	http.Redirect(w, r, authCodeURL, http.StatusFound)
}

func (a *Auth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.SessionStore.Get(r, "session")
	sessionReturn := r.FormValue("return")
	if sessionReturn == "" {
		sessionReturn = r.Header.Get("Referer")
	}
	if sessionReturn == "" {
		sessionReturn = "/"
	}
	delete(session.Values, "identity")
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, sessionReturn, http.StatusFound)
}

func (a *Auth) MeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := a.SessionStore.Get(r, "session")
	identity, ok := session.Values["identity"]
	if !ok {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	dump(w, identity)
}
