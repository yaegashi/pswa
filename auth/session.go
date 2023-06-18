package auth

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	SessionCookieName = "session"
	StateValueName    = "state"
	ReturnValueName   = "return"
	IdentityValueName = "identity"
	DebugValueName    = "debug"
)

type Identity struct {
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

func (a *Auth) Session(r *http.Request) *sessions.Session {
	session, _ := a.SessionStore.Get(r, SessionCookieName)
	return session
}

func (a *Auth) Identity(r *http.Request) *Identity {
	identity, _ := a.Session(r).Values[IdentityValueName].(*Identity)
	return identity
}
