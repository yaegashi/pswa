package core

import (
	"github.com/yaegashi/pswa/auth"
	"github.com/yaegashi/pswa/config"
)

type Core struct {
	Root   string
	Config *config.Config
	Routes []*config.Route
	Auth   *auth.Auth
}

func New(root string, cfg *config.Config, auth *auth.Auth) *Core {
	routes := make([]*config.Route, len(cfg.Routes))
	for i := 0; i < len(routes); i++ {
		routes[i] = cfg.Routes[i]
	}
	return &Core{
		Root:   root,
		Config: cfg,
		Auth:   auth,
		Routes: routes,
	}
}
