package ginx

import (
	"reflect"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	vd9 "gopkg.in/go-playground/validator.v9"
)

var RegexpMap = map[string]*regexp.Regexp{
	"order":   regexp.MustCompile("^[-]?[A-Za-z][A-Za-z0-9_]*$"),
	"chinese": regexp.MustCompile("^[\u4e00-\u9fa5]+$"),
	"name":    regexp.MustCompile("^[\u4e00-\u9fa5A-Za-z0-9_]+$"),
	"path":    regexp.MustCompile("^[\u4e00-\u9fa5A-Za-z0-9_/]+$"),

	// just for chinese mobile phone number
	"mobile": regexp.MustCompile("^[1]([3-9])[0-9]{9}$"),

	// just for chinese telephone number
	"telephone": regexp.MustCompile("^(0\\d{2,3}(\\-)?)?\\d{7,8}$"),
}

type Validator struct {
	*vd9.Validate
}

var _ binding.StructValidator = Validator{}

//使用v9版本，加了一些扩展
func NewValidator() Validator {
	vd := vd9.New()
	for k, v := range RegexpMap {
		vd.RegisterValidation(k, match(v))
	}
	return Validator{vd}
}

// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v Validator) ValidateStruct(obj interface{}) error {
	value := reflect.ValueOf(obj)
	valueType := value.Kind()
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	if valueType == reflect.Struct {
		if err := v.Struct(obj); err != nil {
			return err
		}
	}
	return nil
}

// Engine returns the underlying validator engine which powers the default
// Validator instance. This is useful if you want to register custom validations
// or struct level validations. See validator GoDoc for more info -
// https://godoc.org/gopkg.in/go-playground/validator.v9
func (v Validator) Engine() interface{} {
	return v.Validate
}

func match(r *regexp.Regexp) func(vd9.FieldLevel) bool {
	return func(fl vd9.FieldLevel) bool {
		return r.MatchString(fl.Field().String())
	}
}

func IsValidationError(err error) (ok bool) {
	if err != nil {
		_, ok = err.(vd9.ValidationErrors)
	}
	return
}
