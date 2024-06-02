package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"webapp/templates"
    "webapp/source_map"

	"github.com/gin-gonic/gin"
)


func isDevelopment() bool {
    env := os.Getenv("APP_ENV")
    isProd := strings.EqualFold(env, "PRODUCTION")
    return !isProd
}

func main() {
	app := gin.New()
	app.Use(gin.Logger())

    app.Use(func(c *gin.Context) {
        log.Printf("Global log")
        c.Next()
    })

    app.Use(func(c *gin.Context) {
        c.Next()

        if len(c.Errors) == 0 {
            return
        }

        errorStrings := make([]any, len(c.Errors))
        for i, err := range c.Errors {
            errorStrings[i] = err.JSON()
        }

        c.JSON(400, gin.H{ "errors": errorStrings })
    })

    {
        group := app.Group("local")
        group.Use(func(c *gin.Context) {
            log.Printf("Local log")
            if c.Keys == nil {
                c.Keys = make(map[string]interface{})
            }
            c.Keys["test"] = "Name"
            c.Next()
        })

        group.GET("test", func(c *gin.Context) {
            component := templates.Hello(c.Keys["test"].(string))
            c.Error(fmt.Errorf("test error"))

            if err := component.Render(c.Request.Context(), c.Writer); err != nil {
                c.Error(err)
            }
        })
    }

    app.GET("test", func(c *gin.Context) {
        component := templates.Hello("John")
        if err := component.Render(c.Request.Context(), c.Writer); err != nil {
            c.Error(err)
        }
    })

    source_map.InitSourceMapping(app, isDevelopment())

    app.Run()
}
