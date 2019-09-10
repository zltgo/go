package main

import (
	. "fisheep/conf"
	"log"
	"os"
	"sync"

	"github.com/fvbock/endless"
	"github.com/gin-gonic/gin"
)

func main() {
	// 设置运行模式
	gin.SetMode(Pri.GinMode)
	log.Println("gin mode is ", Pri.GinMode)

	// 初始化用户接口和管理接口
	pub := publicRoutes()
	pri := privateRoutes()

	log.Println("Starting servers...")

	w := sync.WaitGroup{}
	w.Add(2)
	go func() {
		err := endless.ListenAndServe(Pub.HttpAddr, pub)
		if err != nil {
			log.Println(err)
		}
		log.Println("Server on " + Pub.HttpAddr + " stopped")
		w.Done()
	}()
	go func() {
		err := endless.ListenAndServe(Pri.HttpAddr, pri)
		if err != nil {
			log.Println(err)
		}
		log.Println("Server on " + Pri.HttpAddr + " stopped")
		w.Done()
	}()
	w.Wait()
	log.Println("All servers stopped. Exiting.")

	os.Exit(0)
}
