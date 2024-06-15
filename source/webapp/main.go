package main

import (
	"log"
	"os"
	"strings"
	"webapp/source_map"
	lists "webapp/features/lists"
	"github.com/gin-gonic/gin"
    "webapp/stuff"
)

func isDevelopment() bool {
    env := os.Getenv("APP_ENV")
    isProd := strings.EqualFold(env, "PRODUCTION")
    return !isProd
}

func main() {
    appContext, err := stuff.CreateApplicationContext()
    if err != nil {
        log.Fatalf("Error while initializing application: %v", err)
    }

	app := gin.New()
	app.Use(gin.Logger())
    lists.InitListHandler(app, &appContext)
    source_map.Init(app, isDevelopment())
    app.Run()
}
