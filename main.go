package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

const (
	EnvTenantID      = "PSWA_TENANT_ID"
	EnvClientID      = "PSWA_CLIENT_ID"
	EnvClientSecret  = "PSWA_CLIENT_SECRET"
	EnvRedirectURL   = "PSWA_REDIRECT_URI"
	EnvSessionKey    = "PSWA_SESSION_KEY"
	EnvListen        = "PSWA_LISTEN"
	EnvWWWRoot       = "PSWA_WWWROOT"
	EnvConfig        = "PSWA_CONFIG"
	DefaultListen    = ":8080"
	DefaultWWWRoot   = "/home/site/wwwroot"
	DefaultConfig    = "/pswa.config.json"
	FormatAADBaseURL = "https://login.microsoftonline.com/%s/v2.0"
)

type App struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SessionKey   string
	Listen       string
	WWWRootPath  string
	ConfigPath   string
	Config       *PSWAConfig
	Provider     *oidc.Provider
	SessionStore *sessions.CookieStore
	OAuth2Config *oauth2.Config
}

type Identity struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type PSWARoute struct {
	Route        string   `json:"route"`
	AllowedRoles []string `json:"allowedRoles"`
}

type PSWARole struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

type PSWAConfig struct {
	Routes []PSWARoute `json:"routes"`
	Roles  []PSWARole  `json:"roles"`
}

func (c *PSWAConfig) GetRoles(members []string) []string {
	memberMap := map[string]struct{}{}
	for _, m := range members {
		memberMap[m] = struct{}{}
	}
	roles := []string{"authenticated"}
	for _, r := range c.Roles {
		for _, m := range r.Members {
			if _, ok := memberMap[m]; ok {
				roles = append(roles, r.Role)
				break
			}
		}
	}
	sort.Strings(roles)
	return roles
}

func dump(w io.Writer, v interface{}) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func (app *App) AuthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := app.SessionStore.Get(r, "session")
	sessionState, ok := session.Values["state"]
	if !ok {
		http.Error(w, "No state in session", http.StatusBadRequest)
		return
	}
	delete(session.Values, "state")
	ctx := r.Context()
	verifier := app.Provider.Verifier(&oidc.Config{ClientID: app.ClientID})
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
	oauth2Token, err := app.OAuth2Config.Exchange(context.Background(), formCode)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %s", err), http.StatusBadRequest)
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token", http.StatusBadRequest)
		return
	}
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %s", err), http.StatusBadRequest)
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
	identity := Identity{
		Name:  claims.Name,
		Email: claims.Email,
		Roles: app.Config.GetRoles(claims.Groups),
	}
	session.Values["identity"] = identity
	err = session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	dump(w, oauth2Token)
	dump(w, rawIDToken)
	dump(w, claims)
}

func (app *App) AuthLoginHandler(w http.ResponseWriter, r *http.Request) {
	state := uuid.New().String()
	authCodeURL := app.OAuth2Config.AuthCodeURL(state)
	session, _ := app.SessionStore.Get(r, "session")
	session.Values["state"] = state
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, authCodeURL, http.StatusFound)
}

func (app *App) AuthLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := app.SessionStore.Get(r, "session")
	delete(session.Values, "identity")
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Error(w, "Logged out", http.StatusOK)
}

func (app *App) AuthMeHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := app.SessionStore.Get(r, "session")
	identity, ok := session.Values["identity"]
	if !ok {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	dump(w, identity)
}

func (app *App) Main(ctx context.Context) error {
	b, err := ioutil.ReadFile(app.ConfigPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &app.Config)
	if err != nil {
		return err
	}
	baseURL := fmt.Sprintf(FormatAADBaseURL, app.TenantID)
	app.Provider, err = oidc.NewProvider(ctx, baseURL)
	if err != nil {
		return err
	}
	app.SessionStore = sessions.NewCookieStore([]byte(app.SessionKey))
	app.OAuth2Config = &oauth2.Config{
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		RedirectURL:  app.RedirectURL,
		Endpoint:     app.Provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	http.HandleFunc("/.auth/login/aad", app.AuthLoginHandler)
	http.HandleFunc("/.auth/login/aad/callback", app.AuthCallbackHandler)
	http.HandleFunc("/.auth/logout", app.AuthLogoutHandler)
	http.HandleFunc("/.auth/me", app.AuthMeHandler)
	http.Handle("/", http.FileServer(http.Dir(app.WWWRootPath)))
	log.Println("Listening on", app.Listen)
	return http.ListenAndServe(app.Listen, nil)
}

func main() {
	gob.Register(Identity{})
	app := &App{
		TenantID:     os.Getenv(EnvTenantID),
		ClientID:     os.Getenv(EnvClientID),
		ClientSecret: os.Getenv(EnvClientSecret),
		RedirectURL:  os.Getenv(EnvRedirectURL),
		SessionKey:   os.Getenv(EnvSessionKey),
		Listen:       os.Getenv(EnvListen),
		WWWRootPath:  os.Getenv(EnvWWWRoot),
	}
	if app.Listen == "" {
		app.Listen = DefaultListen
	}
	if app.WWWRootPath == "" {
		app.WWWRootPath = DefaultWWWRoot
	}
	if app.ConfigPath == "" {
		app.ConfigPath = DefaultConfig
	}
	err := app.Main(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
