package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/config"
	"golang.org/x/oauth2"
)

const (
	FormatAADBaseURL = "https://login.microsoftonline.com/%s/v2.0"
)

type Auth struct {
	Provider              *oidc.Provider
	Verifier              *oidc.IDTokenVerifier
	OAuth2Config          *oauth2.Config
	OAuth2AuthCodeOptions []oauth2.AuthCodeOption
	Config                *config.Config
	SessionStore          sessions.Store
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
