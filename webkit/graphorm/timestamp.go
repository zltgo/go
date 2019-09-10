package graphorm

import (
	"encoding/json"
	"golang.org/x/exp/errors/fmt"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"io"
	"strconv"
)

// if the type referenced in .gqlgen.yml is a function that returns a marshaller we can use it to encode and decode
// onto any existing go type.
func MarshalTimestamp(t time.Time) graphql.Marshaler {
	return graphql.WriterFunc(func(w io.Writer) {
		//两边没有加引号，表示数字
		io.WriteString(w, strconv.FormatInt(t.Unix(), 10))
	})
}

// Unmarshal{Typename} is only required if the scalar appears as an input. The raw values have already been decoded
// from json into int/float64/bool/nil/map[string]interface/[]interface
// if v is zero not nil, the time.Time also will not be zero.
func UnmarshalTimestamp(v interface{}) (time.Time, error) {
	switch v := v.(type) {
	case json.Number:
		n, err := v.Int64()
		return time.Unix(n, 0), err
	case int64:
		return time.Unix(v, 0), nil
	case float64:
		n, err := floatToInt(v)
		return time.Unix(n, 0), err
	case string:
		//support RFC3339 time string like "2006-01-02T15:04:05.999999999Z07:00"
		return  time.Parse(time.RFC3339, v)
	default:
		return time.Time{}, fmt.Errorf("time should be a unix timestamp, got %T", v)
	}
}

const minInt53 = -2251799813685248
const maxInt53 = 2251799813685247

func floatToInt(f float64) (n int64, err error) {
	n = int64(f)
	if float64(n) == f && n >= minInt53 && n <= maxInt53 {
		return n, nil
	}
	return 0, fmt.Errorf("cann't convert %v to int", f)
}
