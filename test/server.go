package test

import (
	"github.com/antonioalfa22/GoGin-API-REST-Template/cmd/routes"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

var TS *httptest.Server

func CreateServer() {
	if TS == nil {
		gin.SetMode("release")
		r := routes.Setup()
		ts := httptest.NewServer(r)
		// Shut down the server and block until all requests have gone through
		defer ts.Close()
		TS = ts
	}
}

func GetServer() *httptest.Server{
	return TS
}