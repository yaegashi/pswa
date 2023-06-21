package main

import (
	"context"
	"fmt"
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
	EnvTenantID     = "PSWA_TENANT_ID"
	EnvClientID     = "PSWA_CLIENT_ID"
	EnvClientSecret = "PSWA_CLIENT_SECRET"
	EnvRedirectURI  = "PSWA_REDIRECT_URI"
	EnvAuthParams   = "PSWA_AUTH_PARAMS"
	EnvSessionKey   = "PSWA_SESSION_KEY"
	EnvListen       = "PSWA_LISTEN"
	EnvWWWRoot      = "PSWA_WWW_ROOT"
	EnvTestRoot     = "PSWA_TEST_ROOT"
	EnvConfig       = "PSWA_CONFIG"
	DefaultListen   = ":8080"
	DefaultWWWRoot  = "/home/site/wwwroot"
	DefaultTestRoot = "/testroot"
	DefaultConfig   = "pswa.config.json"
)

type App struct {
	SessionStore *sessions.CookieStore
	Config       *config.Config
	Auth         *auth.Auth
	Core         *core.Core
	TenantID     string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthParams   string
	SessionKey   string
	Listen       string
	WWWRootPath  string
	TestRootPath string
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
	loggers := logger.WithOptions(zap.WithCaller(false)).Sugar()

	app.SessionStore = sessions.NewCookieStore([]byte(app.SessionKey))

	configPath := app.ConfigPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(app.WWWRootPath, configPath)
	}
	loggers.Infof("Reading config: %s", configPath)
	app.Config, err = config.New(configPath)
	if err != nil {
		loggers.Errorf("Reading config failed: %s", err)
		app.Config = config.Unconfigured
	}

	app.Auth = auth.New(app.Config, app.SessionStore)
	loggers.Infof("OpenID Connect auth config:")
	loggers.Infof("  TenantID    = %s", app.TenantID)
	loggers.Infof("  ClientID    = %s", app.ClientID)
	loggers.Infof("  RedirectURI = %s", app.RedirectURI)
	loggers.Infof("  AuthParams  = %s", app.AuthParams)

	if app.Auth.EasyAuth {
		loggers.Infof("EasyAuth enabled, skipping OpenID Connect auth config")
	} else if app.TenantID == "" || app.ClientID == "" || app.ClientSecret == "" || app.RedirectURI == "" {
		loggers.Errorf("OpenID Connect auth config missing")
	} else {
		err = app.Auth.ConfigureOIDC(app.TenantID, app.ClientID, app.ClientSecret, app.RedirectURI, app.AuthParams)
		if err != nil {
			loggers.Errorf("OpenID Connect auth config failed: %s", err)
		}
	}

	root := app.WWWRootPath
	if app.Config.TestRoot {
		loggers.Warnf("TestRoot enabled")
		root = app.TestRootPath
	}
	loggers.Infof("Serving from root path %s", root)
	app.Core = core.New(root, app.Config, app.Auth)

	mux := http.NewServeMux()
	app.Auth.RegisterHandlers(mux)

	coreHandler := app.Core.FileHandler
	if app.Config.TestHandler {
		coreHandler = app.Core.TestHandler
	}
	mux.Handle("/", app.Core.NewMiddleware()(http.HandlerFunc(coreHandler)))

	handler := logging.NewMiddleware(logger)(mux)

	loggers.Infof("Serving on %s", app.Listen)

	return http.ListenAndServe(app.Listen, handler)
}

func main() {
	app := &App{
		TenantID:     os.Getenv(EnvTenantID),
		ClientID:     os.Getenv(EnvClientID),
		ClientSecret: os.Getenv(EnvClientSecret),
		RedirectURI:  os.Getenv(EnvRedirectURI),
		AuthParams:   os.Getenv(EnvAuthParams),
		SessionKey:   os.Getenv(EnvSessionKey),
		Listen:       os.Getenv(EnvListen),
		WWWRootPath:  os.Getenv(EnvWWWRoot),
		TestRootPath: os.Getenv(EnvTestRoot),
		ConfigPath:   os.Getenv(EnvConfig),
	}
	if app.Listen == "" {
		app.Listen = DefaultListen
	}
	if app.WWWRootPath == "" {
		app.WWWRootPath = DefaultWWWRoot
	}
	if app.TestRootPath == "" {
		app.TestRootPath = DefaultTestRoot
	}
	if app.ConfigPath == "" {
		app.ConfigPath = DefaultConfig
	}
	err := app.Main(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
