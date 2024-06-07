package main

import (
	"log"
	"os"
	"strings"
	"webapp/source_map"
	"webapp/templates"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func isDevelopment() bool {
    env := os.Getenv("APP_ENV")
    isProd := strings.EqualFold(env, "PRODUCTION")
    return !isProd
}

func main() {
    if (isDevelopment()) {
        gin.SetMode(gin.ReleaseMode)
    }

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

    var counterState = templates.State{}

    renderCounter := func(template templ.Component, c *gin.Context) {
        if err := template.Render(c.Request.Context(), c.Writer); err != nil {
            c.Error(err)
        }
    }

    app.GET("/", func(c *gin.Context) {
        template := templates.Page(&counterState)
        renderCounter(template, c)
    })
    app.POST("/", func(c *gin.Context) {
        c.Request.ParseForm()
        valStr := c.Request.Form.Has("count")
        if valStr {
            counterState.Counter += 1
        }
        template := templates.Counts(&counterState)
        renderCounter(template, c)
    })

    source_map.Init(app, isDevelopment())

    app.Run()
}
