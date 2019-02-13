package wepay

import (
	"crypto/rand"
	"crypto/tls"
	"net/http"
	"time"
)

//随机字符串
func RandString(n int) string {
	const letterStr = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const strLen = byte(len(letterStr))

	runes := make([]byte, n)
	rand.Read(runes)
	for i, b := range runes {
		runes[i] = letterStr[b%strLen]
	}
	return string(runes)
}

// NewHttpsClient 获取双向认证的https客户端
func NewHttpsClient(certPath, keyPath string, timeOut time.Duration) (*http.Client, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	tr := &http.Transport{
		TLSClientConfig: config,
	}

	return &http.Client{
		Transport: tr,
		Timeout:   timeOut,
	}, nil
}
