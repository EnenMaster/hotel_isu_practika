package auth

import (
	"net/http"
	"os"

	"github.com/gorilla/securecookie"
)

var sc = securecookie.New(
	[]byte(os.Getenv("SECURECOOKIE_HASH")),
	[]byte(os.Getenv("SECURECOOKIE_BLOCK")),
)

type Session struct {
	UserID int
	Role   string
}

func Set(w http.ResponseWriter, data Session) error {
	enc, err := sc.Encode("session", data)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    enc,
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func Get(r *http.Request) (Session, bool) {
	c, err := r.Cookie("session")
	if err != nil {
		return Session{}, false
	}
	var s Session
	if err = sc.Decode("session", c.Value, &s); err != nil {
		return Session{}, false
	}
	return s, true
}
