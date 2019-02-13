// Copyright 2011 Dmitry Chestnykh. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package captcha implements generation and verification of image and audio
// CAPTCHAs.
//
// A captcha solution is the sequence of digits 0-9 with the defined length.
// There are two captcha representations: image and audio.
//
// An image representation is a PNG-encoded image with the solution printed on
// it in such a way that makes it hard for computers to solve it using OCR.
//
// An audio representation is a WAVE-encoded (8 kHz unsigned 8-bit) sound with
// the spoken solution (currently in English, Russian, and Chinese). To make it
// hard for computers to solve audio captcha, the voice that pronounces numbers
// has random speed and pitch, and there is a randomly generated background
// noise mixed into the sound.
//
// This package doesn't require external files or libraries to generate captcha
// representations; it is self-contained.
//
// To make captchas one-time, the package includes a memory storage that stores
// captcha ids, their solutions, and expiration time. Used captchas are removed
// from the store immediately after calling Verify or VerifyString, while
// unused captchas (user loaded a page with captcha, but didn't submit the
// form) are collected automatically after the predefined expiration time.
// Developers can also provide custom store (for example, which saves captcha
// ids and solutions in database) by implementing Store interface and
// registering the object with SetCustomStore.
//
// Captchas are created by calling New, which returns the captcha id.  Their
// representations, though, are created on-the-fly by calling WriteImage or
// WriteAudio functions. Created representations are not stored anywhere, but
// subsequent calls to these functions with the same id will write the same
// captcha solution. Reload function will create a new different solution for
// the provided captcha, allowing users to "reload" captcha if they can't solve
// the displayed one without reloading the whole page.  Verify and VerifyString
// are used to verify that the given solution is the right one for the given
// captcha id.
//
// Server provides an http.Handler which can serve image and audio
// representations of captchas automatically from the URL. It can also be used
// to reload captchas.  Refer to Server function documentation for details, or
// take a look at the example in "capexample" subdirectory.
package captcha

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"io"
	"strings"

	. "github.com/zltgo/fileserver/utils"
)

const (
	StdSize = 5
	MaxSize = 32
	// Standard width and height of a captcha image.
	StdWidth  = 240
	StdHeight = 80
)

//新建一个验证码id
func NewStd() string {
	digits := RandBytesMod(StdSize, byte(len(c_strFonts)))
	return Digits2Id(digits)
}

//新建一个验证码id
func New(size int) string {
	Assert(size <= MaxSize, "captcha: size is too big")
	digits := RandBytesMod(size, byte(len(c_strFonts)))
	return Digits2Id(digits)
}

func GetImage(id string, width, height int) (png []byte, err error) {
	var digits []byte
	digits, err = Id2Digits(id)
	if err != nil {
		return
	}
	png = NewImage(id, digits, width, height).encodedPNG()
	return
}

// m_rngKey is a secret key used to deterministically derive seeds for
// PRNGs used in image and audio. Generated once during initialization.
var (
	m_rngKey []byte
	m_block  cipher.Block
)

func init() {
	m_rngKey = RandBytes(32)
	m_block, _ = aes.NewCipher(RandBytes(16))
}

//check id and chars
func Verify(id, chars string) bool {
	if id == "" || chars == "" {
		return false
	}

	digits, err := Id2Digits(id)
	if err != nil {
		return false
	}

	for i := 0; i < len(digits); i++ {
		digits[i] = c_strFonts[digits[i]]
	}

	return strings.ToLower(chars) == strings.ToLower(string(digits))
}

// randomId returns a new random digits and id.
func Digits2Id(digits []byte) string {
	if len(digits) == 0 {
		return ""
	}
	return EncryptUrl(m_block, digits)
}

func Id2Digits(id string) (digits []byte, err error) {
	return DecryptUrl(m_block, id)
}

// Purposes for seed derivation. The goal is to make deterministic PRNG produce
// different outputs for images and audio by using different derived seeds.

// deriveSeed returns a 16-byte PRNG seed from m_rngKey, purpose, id and digits.
// Same purpose, id and digits will result in the same derived seed for this
// instance of running application.
//
//   out = HMAC(m_rngKey, purpose || id || 0x00 || digits)  (cut to 16 bytes)
//
func deriveSeed(purpose byte, id string, digits []byte) (out [16]byte) {
	var buf [sha256.Size]byte
	h := hmac.New(sha256.New, m_rngKey)
	h.Write([]byte{purpose})
	io.WriteString(h, id)
	h.Write([]byte{0})
	h.Write(digits)
	sum := h.Sum(buf[:0])
	copy(out[:], sum)
	return
}
