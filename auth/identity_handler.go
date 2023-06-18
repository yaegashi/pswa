package auth

import (
	"encoding/json"
	"net/http"
)

func (a *Auth) IdentityHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	body := []byte("null")
	if a != nil {
		identity := a.Identity(r)
		b, err := json.Marshal(identity)
		if err == nil {
			body = b
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}
