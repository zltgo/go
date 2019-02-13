package bind

import (
	"regexp"

	"gopkg.in/go-playground/validator.v9"
)

var (
	DefaultValidator *validator.Validate

	regexpMap = map[string]*regexp.Regexp{
		"chinese": regexp.MustCompile("^[\u4e00-\u9fa5]+$"),
		"name":    regexp.MustCompile("^[\u4e00-\u9fa5A-Za-z0-9_]+$"),
		"path":    regexp.MustCompile("^[\u4e00-\u9fa5A-Za-z0-9_/]+$"),

		// just for chinese mobile phone number
		"moblie": regexp.MustCompile("^((\\+86)|(86))?(1(([35][0-9])|[8][0-9]|[7][67]|[4][579]))\\d{8}$"),

		// just for chinese telephone number
		"telephone": regexp.MustCompile("^(0\\d{2,3}(\\-)?)?\\d{7,8}$"),
	}
)

func init() {
	vd := validator.New()

	for k, v := range regexpMap {
		vd.RegisterValidation(k, match(v))
	}
	DefaultValidator = vd
}

func Validate(ptr interface{}) error {
	return DefaultValidator.Struct(ptr)
}

func match(r *regexp.Regexp) func(validator.FieldLevel) bool {
	return func(fl validator.FieldLevel) bool {
		return r.MatchString(fl.Field().String())
	}
}

func IsValidationError(err error) (ok bool) {
	if err != nil {
		_, ok = err.(validator.ValidationErrors)
	}
	return
}
