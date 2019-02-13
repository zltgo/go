// Copyright 2011 Dmitry Chestnykh. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// example of HTTP server that uses the captcha package.
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/zltgo/fileserver/session"

	"github.com/zltgo/fileserver/captcha"
)

var se *session.Session = session.NewStd(0)

var c_strFonts string = "0123456789abcdefghkmnprstuvwxyABCDEFGHJKLMNPQRSTUVXY"

func showFormHandler(w http.ResponseWriter, r *http.Request) {
	//获取session
	values, err := se.Get(r)
	if err != nil {
		fmt.Println(err)
		values, _ = se.Create(r)
	} else {
		digits, err := captcha.Id2Digits(values.Get("captcha"))
		if err != nil {
			fmt.Println("verify: ", err)
		} else {
			for i := 0; i < len(digits); i++ {
				digits[i] = c_strFonts[digits[i]]
			}
			fmt.Println("chars: ", string(digits))
		}

	}

	//保存session
	id := captcha.New(4)
	values.Set("captcha", id)
	se.Set(w, values)

	png, err := captcha.GetImage(id, 200, 70)
	if err != nil {
		fmt.Println("ResponseWriter.Write: ", err)
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(200)
	_, err = w.Write(png)
	if err != nil {
		fmt.Println("ResponseWriter.Write: ", err)
	}
	return
}

func main() {
	http.HandleFunc("/", showFormHandler)
	fmt.Println("Server is at localhost:8666")
	if err := http.ListenAndServe("localhost:8666", nil); err != nil {
		log.Fatal(err)
	}
}
