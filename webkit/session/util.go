package session

import (
	"net"
	"net/http"
	"strings"
	"time"
)

// Cookie stores configuration for a session store.
// Fields are a subset of http.Cookie fields.
type Cookie struct {
	Name   string `default:"_session_id"`
	Path   string `default:"/"`
	Domain string

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'.
	// MaxAge>0 means Max-Age attribute present and given in seconds.
	// Default is 30 days.
	MaxAge int `default:"2592000"`

	// https only if Secure is true
	Secure bool

	// Refuse java script if HttpOnly is true
	HttpOnly bool `default:"true"`
}

func (m *Cookie) GetCookie(r *http.Request) (string, error) {
	ck, err := r.Cookie(m.Name)
	if err != nil || ck.Value == "" {
		return "", http.ErrNoCookie
	}
	return ck.Value, nil
}
func (m *Cookie) SetCookie(w http.ResponseWriter, val string) {
	cookie := &http.Cookie{
		Name:     m.Name,
		Value:    val,
		Path:     m.Path,
		Domain:   m.Domain,
		MaxAge:   m.MaxAge,
		Secure:   m.Secure,
		HttpOnly: m.HttpOnly,
	}
	if m.MaxAge > 0 {
		d := time.Duration(m.MaxAge) * time.Second
		cookie.Expires = time.Now().Add(d)
	} else if m.MaxAge < 0 {
		// Set it to the past to expire now.
		cookie.Expires = time.Unix(1, 0)
	}

	http.SetCookie(w, cookie)
	return
}

func (m *Cookie) DelCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     m.Name,
		Value:    "",
		Path:     m.Path,
		Domain:   m.Domain,
		MaxAge:   -1,
		Secure:   m.Secure,
		HttpOnly: m.HttpOnly,
	}
	// Set it to the past to expire now.
	cookie.Expires = time.Unix(1, 0)

	http.SetCookie(w, cookie)
	return
}

func getIp(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	return ip, err
}
