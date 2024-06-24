package main

import (
	"log"
	createClient "webapp/features/clients/create"
	lists "webapp/features/lists"
	"webapp/source_map"
	"webapp/stuff"

	"github.com/gin-gonic/gin"
)

func main() {
    appContext, err := stuff.CreateApplicationContext()
    if err != nil {
        log.Fatalf("Error while initializing application: %v", err)
    }

	app := gin.New()
	app.Use(gin.Logger())

    lists.InitHandlers(app, &appContext)
    createClient.InitHandler(app, &appContext)
    source_map.Init(app, stuff.IsDevelopment())

    app.Run()
}
