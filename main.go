package main

import (
	"context"
	"encoding/gob"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/sessions"
	"github.com/yaegashi/pswa/auth"
	"github.com/yaegashi/pswa/config"
	"github.com/yaegashi/pswa/core"
	"github.com/yaegashi/pswa/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	DefaultConfig    = "pswa.config.json"
	FormatAADBaseURL = "https://login.microsoftonline.com/%s/v2.0"
)

type App struct {
	SessionStore *sessions.CookieStore
	Config       *config.Config
	Auth         *auth.Auth
	Core         *core.Core
	TenantID     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SessionKey   string
	Listen       string
	WWWRootPath  string
	ConfigPath   string
}

func (app *App) Main(ctx context.Context) error {
	zapCfg := zap.NewDevelopmentConfig()
	zapCfg.EncoderConfig.ConsoleSeparator = " "
	logger, err := zapCfg.Build(zap.AddStacktrace(zapcore.FatalLevel))
	if err != nil {
		return err
	}
	defer logger.Sync()

	gob.Register(&auth.Identity{})

	app.SessionStore = sessions.NewCookieStore([]byte(app.SessionKey))

	configPath := app.ConfigPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(app.WWWRootPath, configPath)
	}
	app.Config, err = config.New(configPath)
	if err != nil {
		return err
	}

	app.Auth, err = auth.New(app.TenantID, app.ClientID, app.ClientSecret, app.RedirectURL, app.Config, app.SessionStore)
	if err != nil {
		return err
	}

	app.Core = core.New(app.WWWRootPath, app.Config, app.SessionStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/.auth/login/aad", app.Auth.LoginHandler)
	mux.HandleFunc("/.auth/login/aad/callback", app.Auth.CallbackHandler)
	mux.HandleFunc("/.auth/logout", app.Auth.LogoutHandler)
	mux.HandleFunc("/.auth/me", app.Auth.MeHandler)
	mux.HandleFunc("/", app.Core.Handler)

	handler := logging.NewMiddleware(logger)(app.Core.NewMiddleware()(mux))

	log.Println("Listening on", app.Listen)

	return http.ListenAndServe(app.Listen, handler)
}

func main() {
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
