package auth

import (
	"net/http"

	"github.com/google/uuid"
)

func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	if !a.EasyAuth && a.OAuth2Config == nil {
		http.Error(w, "OpenID Connect auth config failed: see log output", http.StatusInternalServerError)
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

	if a.EasyAuth {
		http.Redirect(w, r, "/.auth/login/aad?post_login_redirect_uri=/.auth/pswa/easyauth", http.StatusFound)
		return
	}

	authCodeURL := a.OAuth2Config.AuthCodeURL(sessionState, a.OAuth2AuthCodeOptions...)
	http.Redirect(w, r, authCodeURL, http.StatusFound)
}
