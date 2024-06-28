package main

import (
	"log"
	"webapp/features/clients"
	"webapp/features/lists"
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

    stuff.InitErrorHandler(app)
    lists.InitHandlers(app, &appContext)
    clients.InitHandlers(app, &appContext)
    source_map.Init(app, appContext.IsDevelopment())

    app.Run()
}
