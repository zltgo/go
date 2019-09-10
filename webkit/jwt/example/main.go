package main

import (
	"log"
	"net/http"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
	"github.com/zltgo/webkit/jwt"
)

type usrInfo struct {
	ID   string
	Name string
}

type LoginForm struct {
	Name     string `binding:"alphanum,min=5,max=32"`
	Password string `binding:"alphanum,min=5,max=32"`
}

func main() {
	//use default TokenGetter, witch get token in http.Request.Header by "ACCESS-TOKEN"
	auth := jwt.NewAuth(jwt.AuthOpts{
		HashKey:  "somestringveryveryveryverysecret",
		BlockKey: "somestringveryveryveryverysecret",
	})

	r := gin.Default()

	r.GET("/login", func(c *gin.Context) {
		lf := LoginForm{}
		if err := c.ShouldBind(&lf); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		usr := usrInfo{
			ID:   "111", //get from db
			Name: lf.Name,
		}

		tk, err := auth.NewAuthToken(c.Request, usr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tk)
	})

	r.GET("/me", func(c *gin.Context) {
		usr := usrInfo{}
		if err := auth.GetAccessInfo(c.Request, &usr); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, usr)
	})

	// Listen and Server in http://0.0.0.0:8080
	log.Println(endless.ListenAndServe(":8080", r))
}
