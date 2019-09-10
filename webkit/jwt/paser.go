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
	"time"
)

var (
	//errors
	ErrMacInvalid  = errors.New("jwt: token is not valid")
	ErrDecrypt     = errors.New("jwt: token could not be decrypted")
	ErrExpired     = errors.New("jwt: token is expired")
	ErrIssued      = errors.New("jwt: token used before issued")
	ErrNoTimestamp = errors.New("jwt: timestamps not found")

	// TimeFunc provides the current time when parsing token to validate "exp" claim (expiration time).
	// You can override it to use another time value.  This is useful for testing or if your
	// server uses a different time zone than your tokens.
	TimeNow = time.Now
)

type Parser interface {
	CreateToken(tokenValues interface{}) (string, error)

	//parse a token and return the values.
	ParseToken(tk string, pTokenValues interface{}) error

	//return the max-age attribute in seconds
	MaxAge() int
}

var _ Parser = &paser{}

//parse or new a token
type paser struct {
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

// New returns a new paser for token.
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
	rv := &paser{hashKey: hashKey, maxAge: maxAge}

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
func (m *paser) CreateToken(tokenValues interface{}) (string, error) {
	// It may lose precision when parsing token.
	// Because numbers are converted to float64 when decoding json.
	tv := struct {
		TokenValues interface{}
		ExpiresAt   int64 `json:",omitempty"`
		IssuedAt    int64
	}{
		TokenValues: tokenValues,
		IssuedAt:    TimeNow().Unix(),
	}

	if m.maxAge > 0 {
		tv.ExpiresAt = tv.IssuedAt + int64(m.maxAge)
	}
	//encode json
	b, err := json.Marshal(&tv)
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

// parse a token and return the values.
// pTokenValues supposed to have point type.
func (m *paser) ParseToken(tk string, pTokenValues interface{}) error {
	//url decode
	b, err := base64.URLEncoding.DecodeString(tk)
	if err != nil {
		return errors.New("jwt: " + err.Error())
	}

	index := len(b) - m.hashSize
	if index <= 0 {
		return ErrMacInvalid
	}
	data := b[:index]
	mac := b[index:]

	//verify mac, mac must create here for thread-safe
	hash := hmac.New(m.hashFunc, m.hashKey)
	if !VerifyMac(hash, data, mac) {
		return ErrMacInvalid
	}
	//decrypt
	if m.block != nil {
		data = Decrypt(m.block, data)
		if data == nil {
			return ErrDecrypt
		}
	}
	//decode json
	tv := struct {
		TokenValues interface{}
		ExpiresAt   int64 `json:",omitempty"`
		IssuedAt    int64
	}{
		TokenValues: pTokenValues,
	}

	if err = json.Unmarshal(data, &tv); err != nil {
		return errors.New("jwt: " + err.Error())
	}

	//check timestamp
	now := TimeNow().Unix()
	if tv.IssuedAt == 0 {
		return ErrNoTimestamp
	}
	if tv.IssuedAt > now {
		return ErrIssued
	}

	if m.maxAge > 0 {
		if tv.ExpiresAt == 0 {
			return ErrNoTimestamp
		}
		if tv.ExpiresAt < now {
			return ErrExpired
		}
	}

	return nil
}

//return the max-age attribute in seconds
func (m *paser) MaxAge() int {
	return m.maxAge
}
