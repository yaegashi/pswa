package auth

import (
	"net/http"

	"github.com/google/uuid"
)

func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if a == nil {
		http.Error(w, "Auth config failed: see log output", http.StatusInternalServerError)
		return
	}
	sessionState := uuid.New().String()
	sessionReturn := r.FormValue(ReturnValueName)
	if sessionReturn == "" {
		sessionReturn = r.Header.Get("Referer")
	}
	sessionDebug := r.FormValue(DebugValueName)
	session := a.Session(r)
	session.Values[StateValueName] = sessionState
	session.Values[ReturnValueName] = sessionReturn
	session.Values[DebugValueName] = sessionDebug
	err := session.Save(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	authCodeURL := a.OAuth2Config.AuthCodeURL(sessionState, a.OAuth2AuthCodeOptions...)
	http.Redirect(w, r, authCodeURL, http.StatusFound)
}
