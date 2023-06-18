package auth

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/config"
	"golang.org/x/oauth2"
)

const (
	FormatAADBaseURL           = "https://login.microsoftonline.com/%s/v2.0"
	EasyAuthAppSettingsEnvName = "WEBSITE_AUTH_ENABLED"
)

type Auth struct {
	Provider              *oidc.Provider
	Verifier              *oidc.IDTokenVerifier
	OAuth2Config          *oauth2.Config
	OAuth2AuthCodeOptions []oauth2.AuthCodeOption
	Config                *config.Config
	SessionStore          sessions.Store
	EasyAuth              bool
}

func New(cfg *config.Config, ss sessions.Store) *Auth {
	return &Auth{
		Config:       cfg,
		SessionStore: ss,
		EasyAuth:     strings.ToLower(os.Getenv(EasyAuthAppSettingsEnvName)) == "true",
	}
}

func (a *Auth) ConfigureOIDC(tenantID, clientID, clientSecret, redirectURI, authParams string) error {
	baseURL := fmt.Sprintf(FormatAADBaseURL, tenantID)
	provider, err := oidc.NewProvider(context.Background(), baseURL)
	if err != nil {
		return err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
	var authCodeOptions []oauth2.AuthCodeOption
	for _, p := range strings.Split(authParams, "&") {
		s := strings.SplitN(p, "=", 2)
		if len(s) == 2 {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam(s[0], s[1]))
		}
	}
	a.Provider = provider
	a.Verifier = verifier
	a.OAuth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "User.Read"},
	}
	a.OAuth2AuthCodeOptions = authCodeOptions
	return nil
}
