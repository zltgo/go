package bind

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type FooBar struct {
	Foo   string `validate:"alphanum"`
	Bar   string `validate:"alphanum"`
	Alice string `json:",omitempty" default:"alice"`
	Bob   string `json:",omitempty"`
}

func TestBind(t *testing.T) {
	extra := map[string][]string{"Bob": []string{"bob"}}

	Convey("Bind successfully", t, func() {
		Convey("test GetType by form", func() {
			req, err := http.NewRequest("GET", "/?Foo=ping&Bar=pong", nil)
			So(err, ShouldBeNil)

			v, err := GetType(reflect.TypeOf(FooBar{}), req, extra)
			So(err, ShouldBeNil)

			fb, ok := v.Interface().(FooBar)
			So(ok, ShouldBeTrue)
			So(fb.Alice+fb.Bob+fb.Foo+fb.Bar, ShouldEqual, "alicebobpingpong")
		})

		Convey("test Bind by form", func() {
			req, err := http.NewRequest("GET", "/?Foo=ping&Bar=pong", nil)
			So(err, ShouldBeNil)

			var fb FooBar
			err = Bind(&fb, req, extra)
			So(err, ShouldBeNil)
			So(fb.Alice+fb.Bob+fb.Foo+fb.Bar, ShouldEqual, "alicebobpingpong")
		})

		Convey("by json", func() {
			input := FooBar{
				Foo: "ping",
				Bar: "pong",
			}
			b, _ := json.Marshal(input)
			req, _ := http.NewRequest("POST", "/", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json; charset=utf-8")

			v, err := GetType(reflect.TypeOf(&FooBar{}), req, extra)
			So(err, ShouldBeNil)

			fb, ok := v.Interface().(*FooBar)
			So(ok, ShouldBeTrue)
			So(fb.Alice+fb.Bob+fb.Foo+fb.Bar, ShouldEqual, "alicebobpingpong")
		})
	})

	Convey("Bind with error", t, func() {
		input := FooBar{
			Foo: "ping",
			Bar: "pong",
		}
		b, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/", bytes.NewReader(b))

		_, err := GetType(reflect.TypeOf(FooBar{}), req, nil)
		So(err, ShouldNotBeNil)
	})
}

func Benchmark_GetType(b *testing.B) {
	extra := map[string][]string{"Bob": []string{"bob"}}
	req, err := http.NewRequest("GET", "/?Foo=ping&Bar=pong", nil)
	if err != nil {
		b.Error(err)
		return
	}

	_, err = GetType(reflect.TypeOf(FooBar{}), req, extra)
	if err != nil {
		b.Error(err)
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetType(reflect.TypeOf(FooBar{}), req, extra)
	}
}

func Benchmark_Bind(b *testing.B) {
	extra := map[string][]string{"Bob": []string{"bob"}}
	req, err := http.NewRequest("GET", "/?Foo=ping&Bar=pong", nil)
	if err != nil {
		b.Error(err)
		return
	}

	var fb FooBar
	err = Bind(&fb, req, extra)
	if err != nil {
		b.Error(err)
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var fb FooBar
		Bind(&fb, req, extra)
	}
}
