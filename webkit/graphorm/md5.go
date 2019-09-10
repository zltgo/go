package graphorm

import (
	"crypto/md5"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
)

//16 bytes in db and 32 length as hex string in struct field
type MD5 string

// New MD5 returns a new unique  MD5.
func NewMD5(input string) MD5 {
	t := md5.New()
	io.WriteString(t, input)
	return MD5(hex.EncodeToString(t.Sum(nil)))
}

//check  MD5 is valid or not
func (m MD5) IsValid() bool {
	if len(m) != 32 {
		return false
	}

	for _, c := range m {
		if ok := ('0' <= c && c <= '9') ||
			('a' <= c && c <= 'f') ||
			('A' <= c && c <= 'F'); !ok {
			return false
		}
	}
	return true
}

//save to db
func (m MD5) Value() (driver.Value, error) {
	if len(m) != 32 {
		return nil, fmt.Errorf("invalid  MD5")
	}
	return hex.DecodeString(string(m))
}

//from db
func (m *MD5) Scan(src interface{}) error {
	switch src := src.(type) {
	case MD5:
		*m = src
		return nil
	case []byte:
		if len(src) != 16 {
			return fmt.Errorf("invalid  MD5")
		}
		*m = MD5(hex.EncodeToString(src))
		return nil
	case string:
		if len(src) != 16 {
			return fmt.Errorf("invalid  MD5")
		}
		*m = MD5(hex.EncodeToString([]byte(src)))
		return nil
	}
	return fmt.Errorf("cannot convert %T to  MD5", src)
}

// format
func (m MD5) String() string {
	return string(m)
}
