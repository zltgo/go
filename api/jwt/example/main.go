package main

import (
	"log"

	"github.com/fvbock/endless"
	"github.com/zltgo/api"
	"github.com/zltgo/api/jwt"
)

func helloHandler(uid jwt.UID) string {
	return "hello, your UserId is " + string(uid)
}

type LoginForm struct {
	Name     string `validate:"alphanum,min=5,max=32"`
	Password string `validate:"alphanum,min=5,max=32"`
}

func login(lf LoginForm) (int, string) {
	if lf.Name == "admin" && lf.Password == "admin" {
		return 200, "9527"
	} else {
		return 401, "name or password error"
	}
}

func main() {
	router := api.Default()

	//Token lifetime is 30 days.
	//If your server is not https, you neet a blockkey for secret.
	accessTp := jwt.NewParser(1800, []byte("somestringsecret"), nil)
	refreshTp := jwt.NewParser(30*86400, []byte("somestringveryveryveryverysecret"), nil)

	//use default TokenGetter, witch get token in http.Request.Header by "ACCESS-TOKEN"
	auth := jwt.NewAuth(accessTp, refreshTp, nil)

	// POST:/token can be used by clients to create a auth token.
	// Clients need to put it in request header named  "ACCESS-TOKEN".
	router.POST("/login", auth.LoginHandler(login))

	// Clienst need to put refresh token in request header named  "REFRESH-TOKEN".
	router.GET("/token", auth.RefreshHandler)

	//AuthHandler, used to get the userId stored in token and put it in context with UserIdKey.
	mw := api.NewMiddware(auth.AuthHandler)
	router.GET("/hello", mw.Then(helloHandler)...)
	//some other handlers...

	log.Println(endless.ListenAndServe(":8111", router))
}
