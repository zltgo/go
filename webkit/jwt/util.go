package jwt

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"hash"
	"hash/fnv"
	"io"
)

// Authentication -------------------------------------------------------------
// verifyMac verifies that a message authentication code (MAC) is valid.
func VerifyMac(h hash.Hash, value []byte, mac []byte) bool {
	h.Write(value)
	mac2 := h.Sum(nil)
	// Check that both MACs are of equal length, as subtle.ConstantTimeCompare
	// does not do this prior to Go 1.4.
	if len(mac) == len(mac2) && subtle.ConstantTimeCompare(mac, mac2) == 1 {
		return true
	}
	return false
}

// Encryption -----------------------------------------------------------------

// encrypt encrypts a value using the given block in counter mode.
//
// A random initialization vector (http://goo.gl/zF67k) with the length of the
// block size is prepended to the resulting ciphertext.
func Encrypt(block cipher.Block, value []byte) []byte {
	iv := RandBytes(block.BlockSize())
	// Encrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)
	// Return iv + ciphertext.
	return append(iv, value...)
}

// Decrypt decrypts a value using the given block in counter mode.
//
// The value to be decrypted must be prepended by a initialization vector
// (http://goo.gl/zF67k) with the length of the block size.
// If failed to decrypt, return nil.
func Decrypt(block cipher.Block, value []byte) []byte {
	size := block.BlockSize()
	if len(value) > size {
		// Extract iv.
		iv := value[:size]
		// Extract ciphertext.
		value = value[size:]
		// Decrypt it.
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)
		return value
	}
	return nil
}

// randomBytes returns a byte slice of the given size read from CSPRNG.
func RandBytes(size int) (b []byte) {
	b = make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic("RandBytes: error reading random source: " + err.Error())
	}
	return
}

//随机字符号串
func RandString(n int) string {
	const letterStr = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const lsLen = len(letterStr)

	var runes = make([]byte, n)
	rand.Read(runes)
	for i, b := range runes {
		runes[i] = letterStr[b%byte(lsLen)]
	}
	return string(runes)
}

func Hash64(input string) string {
	h := fnv.New64()
	io.WriteString(h, input)
	return hex.EncodeToString(h.Sum(nil))
}
