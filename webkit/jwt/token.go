package jwt

import (
	"errors"
	"fmt"
	"reflect"
	"net/http"
)

var (
	//errors
	ErrNoToken = errors.New("jwt: no token present in request")
)

// Interface for extracting a token from an HTTP request.
// The GetToken method should return a token string or an error.
// If no token is present, you can return ErrNoTokenInRequest.
type TokenGetter interface {
	GetToken(*http.Request) (string, error)
}

// Extractor for finding a token in a header.  Looks at each specified
// header in order until there's a match
type HeaderGetter []string

func (m HeaderGetter) GetToken(r *http.Request) (string, error) {
	// loop over header names and return the first one that contains data
	for _, header := range m {
		if ah := r.Header.Get(header); ah != "" {
			return ah, nil
		}
	}
	return "", ErrNoToken
}

type Token struct {
	TokenGetter
	Parser
	GrantType string
}

type TokenValue struct {
	AgentHash string `json:",omitempty"`
	GrantType string `json:",omitempty"`
	Metadata  interface{}
}

// Get values stored in token.
// pMeta supposed to have point type.
func (t Token) DecodeValues(r *http.Request, pMeta interface{}) error {
	if reflect.TypeOf(pMeta).Kind() != reflect.Ptr {
		panic("input parameter supposed to have point type")
	}

	//get token string
	token, err := t.GetToken(r)
	if err != nil {
		return err
	}
	//parse token
	tv := TokenValue{Metadata: pMeta}
	if err = t.ParseToken(token, &tv); err != nil {
		return err
	}

	// validate grant type and user agent
	if tv.GrantType != t.GrantType {
		return fmt.Errorf("jwt: grant type mismatched: expected %s, got %s", t.GrantType, tv.GrantType)
	}
	if tv.AgentHash != Hash64(r.UserAgent()) {
		return fmt.Errorf("jwt: user agent mismatched: %s", r.UserAgent())
	}

	return nil
}

// Set values to token string.
func (t Token) EncodeValues(r *http.Request, meta interface{}) (string, error) {
	return t.CreateToken(&TokenValue{
		AgentHash: Hash64(r.UserAgent()),
		GrantType: t.GrantType,
		Metadata:  meta,
	})
}
