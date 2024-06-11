package main

import (
    "common/models/foaie"
	"os"
	"strings"
	"webapp/source_map"
	template_test "webapp/templates/test"
	template_lists "webapp/templates/lists"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/currency"
)

func isDevelopment() bool {
    env := os.Getenv("APP_ENV")
    isProd := strings.EqualFold(env, "PRODUCTION")
    return !isProd
}

func renderTemplate(template templ.Component, c *gin.Context) {
    if err := template.Render(c.Request.Context(), c.Writer); err != nil {
        c.Error(err)
    }
}

func main() {
	app := gin.New()
	app.Use(gin.Logger())

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

    var counterState = template_test.State{}


    app.GET("/", func(c *gin.Context) {
        template := template_test.Page(&counterState)
        renderTemplate(template, c)
    })
    app.POST("/", func(c *gin.Context) {
        c.Request.ParseForm()
        valStr := c.Request.Form.Has("count")
        if valStr {
            counterState.Counter += 1
        }
        template := template_test.Counts(&counterState)
        renderTemplate(template, c)
    })
    app.GET("/lists", func(c *gin.Context) {
        filteredLists := template_lists.FilteredLists{
            Values: []foaie.Foaie{
                {
                    Id: 1,
                    Pret: 200,
                    Tip: "Mare",
                    ProvidedTransport: true,
                    Hotel: "test",
                },
                {
                    Id: 2,
                    Pret: 100,
                    Tip: "Excursie",
                    ProvidedTransport: false,
                    Hotel: "test test test test test",
                },
            },
            FieldsShouldRender: foaie.AllFieldMask(),
            CurrencyFormatter: currency.ISO.Default(currency.EUR),
        }
        template := template_lists.Page(&filteredLists)
        renderTemplate(template, c)
    })

    source_map.Init(app, isDevelopment())

    app.Run()
}
