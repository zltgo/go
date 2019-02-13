package bind

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"net/url"
	"reflect"

	"github.com/zltgo/reflectx"
)

const (
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMEXML2              = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
)

// Like Bind, Create a struct or structPtr  by Type t.
// It panics if t is not a struct or structPtr type.
func GetType(t reflect.Type, r *http.Request, params map[string][]string) (reflect.Value, error) {
	rv := reflectx.AllocDefault(t)
	switch {
	case t.Kind() == reflect.Struct:
		if err := bindWithParams(r, rv.Addr().Interface(), params); err != nil {
			return reflect.Value{}, err
		}
		return rv, DefaultValidator.Struct(rv.Addr().Interface())
	case t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct:
		if err := bindWithParams(r, rv.Interface(), params); err != nil {
			return reflect.Value{}, err
		}
		return rv, DefaultValidator.Struct(rv.Interface())
	default:
		panic("expect struct or struct pointer, got " + t.String())
	}
}

// Reassign each field in the struct  that is tagged with  'default', 'json', 'xml' and 'form'.
// Then validate each field that is tagged with 'validate'.
// Note that, json.NewDecoder is different from form,  it will set field to zero
// if the string is empty. Don't forget to set omitempty tag.
func Bind(ptr interface{}, r *http.Request, params map[string][]string) error {
	// set default value
	reflectx.SetDefault(ptr)

	// bind value from http.Request and parameters
	if err := bindWithParams(r, ptr, params); err != nil {
		return err
	}

	return DefaultValidator.Struct(ptr)
}

// Just bind value from http.Request and parameters.
func bindWithParams(r *http.Request, ptr interface{}, params map[string][]string) error {
	//bind values from http.Request
	mime := GetContentType(r.Header)
	if r.Method == "GET" || r.Method == "DELETE" {
		mime = MIMEPOSTForm
	}

	switch mime {
	case MIMEPOSTForm, MIMEMultipartPOSTForm:
		err := r.ParseForm()
		if err != nil {
			return err
		}
		// you'd better to call this by your self
		//r.ParseMultipartForm(32 << 10) // 32 MB

		//combine form and params
		if len(r.Form) > 0 && params == nil {
			params = make(map[string][]string, len(r.Form))
		}
		for k, v := range r.Form {
			params[k] = v
		}
		return reflectx.FormToStruct(params, ptr)
	case MIMEJSON:
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(ptr); err != nil {
			return err
		}
	case MIMEXML, MIMEXML2:
		decoder := xml.NewDecoder(r.Body)
		if err := decoder.Decode(ptr); err != nil {
			return err
		}
	default:
		return errors.New("context type not support: " + mime)
	}

	//bind params
	if len(params) > 0 {
		return reflectx.FormToStruct(params, ptr)
	}
	return nil
}

func GetContentType(h http.Header) string {
	content := url.Values(h).Get("Content-Type")
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}
