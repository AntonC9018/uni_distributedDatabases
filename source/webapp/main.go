package main

import (
	"log"
	"webapp/source_map"
	lists "webapp/features/lists"
	"github.com/gin-gonic/gin"
    "webapp/stuff"
)

func main() {
    appContext, err := stuff.CreateApplicationContext()
    if err != nil {
        log.Fatalf("Error while initializing application: %v", err)
    }

	app := gin.New()
	app.Use(gin.Logger())

    lists.InitListHandler(app, &appContext)
    source_map.Init(app, stuff.IsDevelopment())

    app.Run()
}
