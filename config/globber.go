package config

import (
	"fmt"
	"strings"

	"github.com/gobwas/glob"
)

type Globber struct {
	Prefix string
	Leaf   string
	Glob   glob.Glob
}

func (gl *Globber) Compile(route string) error {
	if len(route) == 0 || route[0] != '/' {
		return fmt.Errorf("Route %q non-absolute path", route)
	}
	n := strings.LastIndexByte(route, '/')
	g, err := glob.Compile(route[n:])
	if err != nil {
		return fmt.Errorf("Route %q bad glob pattern: %w", route, err)
	}
	gl.Prefix = route[:n]
	gl.Leaf = route[n:]
	gl.Glob = g
	return nil
}

func (gl *Globber) StripPrefix(path string) string {
	return strings.TrimPrefix(path, gl.Prefix)
}

func (gl *Globber) Match(path string) bool {
	if !strings.HasPrefix(path, gl.Prefix) {
		return false
	}
	n := len(gl.Prefix)
	if gl.Leaf == "/*" {
		return len(path) == n || path[n] == '/'
	}
	return gl.Glob.Match(path[n:])
}
