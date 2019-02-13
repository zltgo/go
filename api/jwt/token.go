package jwt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash"
	"net/http"
	"time"

	"github.com/zltgo/reflectx/values"
)

var (
	//errors
	ErrMacInvalid       = errors.New("jwt: token is not valid")
	ErrDecrypt          = errors.New("jwt: token could not be decrypted")
	ErrExpired          = errors.New("jwt: token is expired")
	ErrTimestamp        = errors.New("jwt: token used before issued")
	ErrNoTokenInRequest = errors.New("jwt: no token present in request")

	CreateTimeKey = "_ct"
	// TimeFunc provides the current time when parsing token to validate "exp" claim (expiration time).
	// You can override it to use another time value.  This is useful for testing or if your
	// server uses a different time zone than your tokens.
	TimeNow = time.Now

	// Default TokenParser
	DefaultParser = NewParser(30*86400, nil, nil)
)

type Parser interface {
	CreateToken(map[string]interface{}) (string, error)

	//parse a token and return the values.
	ParseToken(string) (map[string]interface{}, error)

	//return the max-age attribute in seconds
	MaxAge() int
}

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
	return "", ErrNoTokenInRequest
}

var _ Parser = &tokenParser{}

//parse or new a token
type tokenParser struct {
	// maxAge<=0 means no 'Max-Age' attribute specified.
	// maxAge>0 means Max-Age attribute present and given in seconds.
	maxAge int

	// hashKey is used to authenticate values using HMAC.
	//It is recommended to use a key with 16 , 32, 48 or 64 bytes
	// to select sha1, sha256, sha384, or sha512.
	// If hashKey is nil, it will be created by RandBytes(16).
	hashKey  []byte
	hashFunc func() hash.Hash
	hashSize int

	// blockKey is used to encrypt values.
	//The key length must correspond to the block size
	// of the encryption algorithm. For AES, used by default, valid lengths are
	// 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
	// if blockKey is nil,  values will not  encrypt.
	block cipher.Block
}

// New returns a new tokenParser for token.
//
// maxAge<=0 means no 'Max-Age' attribute specified.
// maxAge>0 means Max-Age attribute present and given in seconds.
// NewTokenParser set maxAge to default 30 days when you input 0 maxAge.
//
// Note that keys created using RandBytes() are not automatically
// persisted. New keys will be created when the application is restarted, and
// previously issued tokens will not be able to be decoded.
func NewParser(maxAge int, hashKey, blockKey []byte) Parser {
	if len(hashKey) == 0 {
		hashKey = RandBytes(16)
	}
	rv := &tokenParser{hashKey: hashKey, maxAge: maxAge}

	if len(blockKey) > 0 {
		if block, err := aes.NewCipher(blockKey); err == nil {
			rv.block = block
		} else {
			panic(err)
		}
	}

	n := len(hashKey)
	switch {
	case n < 32:
		rv.hashFunc = sha1.New
		rv.hashSize = sha1.Size
	case n < 48:
		rv.hashFunc = sha256.New
		rv.hashSize = sha256.Size
	case n < 64:
		rv.hashFunc = sha512.New384
		rv.hashSize = sha512.Size384
	default:
		rv.hashFunc = sha512.New
		rv.hashSize = sha512.Size
	}

	return rv
}

//create a token for the given values, thread-safe
func (m *tokenParser) CreateToken(mp map[string]interface{}) (string, error) {
	if m.maxAge > 0 {
		// It may lose precision when parsing token.
		// Because numbers are converted to float64 when decoding json.
		mp[CreateTimeKey] = TimeNow().Unix()
	}
	//encode json
	b, err := json.Marshal(mp)
	if err != nil {
		return "", errors.New("jwt: " + err.Error())
	}
	//encrypt
	if m.block != nil {
		b = Encrypt(m.block, b)
	}
	//add mac to tail, mac must create here for thread-safe
	hash := hmac.New(m.hashFunc, m.hashKey)
	hash.Write(b)

	return base64.URLEncoding.EncodeToString(hash.Sum(b)), nil
}

//parse a token and return the values.
//the create time of the token is also included in the return map by CreateTimeKey.
func (m *tokenParser) ParseToken(tk string) (map[string]interface{}, error) {
	//url decode
	b, err := base64.URLEncoding.DecodeString(tk)
	if err != nil {
		return nil, errors.New("jwt: " + err.Error())
	}

	index := len(b) - m.hashSize
	if index <= 0 {
		return nil, ErrMacInvalid
	}
	data := b[:index]
	mac := b[index:]

	//verify mac, mac must create here for thread-safe
	hash := hmac.New(m.hashFunc, m.hashKey)
	if !VerifyMac(hash, data, mac) {
		return nil, ErrMacInvalid
	}
	//decrypt
	if m.block != nil {
		data = Decrypt(m.block, data)
		if data == nil {
			return nil, ErrDecrypt
		}
	}
	//decode json
	jsonMap := values.JsonMap{}
	if err = jsonMap.Decode(data); err != nil {
		return nil, errors.New("jwt: " + err.Error())
	}

	//check timestamp
	if m.maxAge > 0 {
		t := jsonMap.ValueOf(CreateTimeKey).Int64()
		if t == 0 {
			return nil, ErrTimestamp
		}
		d := TimeNow().Unix() - t
		if d < 0 {
			return nil, ErrTimestamp
		}
		if d > int64(m.maxAge) {
			return jsonMap, ErrExpired
		}
	}

	return jsonMap, nil
}

//return the max-age attribute in seconds
func (m *tokenParser) MaxAge() int {
	return m.maxAge
}
