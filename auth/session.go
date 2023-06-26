package auth

import (
	"encoding/gob"
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionCookieName = "PSWASession"
	StateValueName    = "state"
	ReturnValueName   = "return"
	IdentityValueName = "identity"
	DebugValueName    = "debug"
)

type Identity struct {
	Typ   string   `json:"typ,omitempty"`
	Id    string   `json:"id,omitempty"`
	Name  string   `json:"name,omitempty"`
	Email string   `json:"email,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

func init() {
	gob.Register(&Identity{})
}

func (a *Auth) Session(r *http.Request) *sessions.Session {
	session, _ := a.SessionStore.Get(r, SessionCookieName)
	session.Options.HttpOnly = true
	session.Options.Secure = true
	session.Options.SameSite = http.SameSiteNoneMode
	return session
}

func (a *Auth) Identity(r *http.Request) *Identity {
	identity, _ := a.Session(r).Values[IdentityValueName].(*Identity)
	return identity
}
