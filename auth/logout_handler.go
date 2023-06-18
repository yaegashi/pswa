package auth

import "net/http"

func (a *Auth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	sessionReturn := r.FormValue(ReturnValueName)
	if sessionReturn == "" {
		sessionReturn = r.Header.Get("Referer")
	}
	if sessionReturn == "" {
		sessionReturn = "/"
	}
	if a != nil {
		session := a.Session(r)
		session.Options.MaxAge = -1
		err := session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.SetCookie(w, &http.Cookie{Name: SessionCookieName, Path: "/", MaxAge: -1})
	}
	http.Redirect(w, r, sessionReturn, http.StatusFound)
}
