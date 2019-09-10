package graphorm

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"

	"gopkg.in/mgo.v2/bson"
)

//12 bytes in db and 24 length as hex string in struct field
type UUID string

type Model struct {
	ID        UUID       `json:"id" sql:"type:char(12);primary_key" validate:"hexadecimal,len=24"`
	CreatedAt time.Time  `json:"createdAt" `
	UpdatedAt time.Time  `json:"updatedAt" `
	DeletedAt *time.Time `json:"deletedAt" sql:"index"`
}

func (m *Model) BeforeCreate(scope *gorm.Scope) {
	// new UUID only if ID is nil
	if scope.PrimaryKeyZero() {
		scope.SetColumn("id", NewUUID())
	}
}

// NewUUID returns a new unique UUID.
func NewUUID() UUID {
	return UUID(bson.NewObjectId().Hex())
}

func ZeroUUID() UUID {
	return UUID("000000000000000000000000")
}

//check uuid is valid or not
func (u UUID) IsValid() bool {
	if len(u) != 24 {
		return false
	}

	for _, c := range u {
		if ok := ('0' <= c && c <= '9') ||
			('a' <= c && c <= 'f') ||
			('A' <= c && c <= 'F'); !ok {
			return false
		}
	}
	return true
}

//save to db
func (u UUID) Value() (driver.Value, error) {
	if len(u) != 24 {
		return nil, fmt.Errorf("invalid UUID: %s", string(u))
	}
	return hex.DecodeString(string(u))
}

//from db
func (u *UUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case UUID:
		*u = src
		return nil
	case []byte:
		if len(src) != 12 {
			return fmt.Errorf("invalid UUID")
		}
		*u = UUID(hex.EncodeToString(src))
		return nil
	case string:
		if len(src) != 12 {
			return fmt.Errorf("invalid UUID")
		}
		*u = UUID(hex.EncodeToString([]byte(src)))
		return nil
	}
	return fmt.Errorf("cannot convert %T to UUID", src)
}

// UnmarshalGQL implements the graphql.Marshaler interface
func (u *UUID) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot convert %T to UUID", v)
	}

	uuid := UUID(str)
	if !uuid.IsValid() {
		return fmt.Errorf("invalid UUID: %s", str)
	}
	*u = uuid
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
// Add quote to make sure UUID has string type in json.
func (u UUID) MarshalGQL(w io.Writer) {
	w.Write([]byte(strconv.Quote(string(u))))
}

// format
func (u UUID) String() string {
	return string(u)
}
