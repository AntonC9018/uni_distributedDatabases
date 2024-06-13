package main

import (
	"common/config"
	database_config "common/database"
	"common/models/foaie"
	"fmt"
	"log"
	"os"
	"strings"
	database "webapp/database"
	"webapp/source_map"
	template_lists "webapp/templates/lists"

	"github.com/go-ozzo/ozzo-validation"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"golang.org/x/text/currency"
	"gorm.io/gorm"
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

type ListsQuery struct {
    Database string `query:"db"`
}

type Pagination struct {
    Cursor paginator.Cursor
    Order paginator.Order
    Limit int
}

type ErrorScope struct {
    beforeCount int
    context *gin.Context
}
func CreateErrorScope(context *gin.Context) ErrorScope {
    return ErrorScope{
        beforeCount: len(context.Errors),
        context: context,
    }
}
func (s *ErrorScope) HasErrors() bool {
    return s.beforeCount > len(s.context.Errors)
}

func GetPagination(c *gin.Context) Pagination {
    errScope := CreateErrorScope(c)
    ret := Pagination{}

    err := c.BindQuery(&ret.Cursor)
    if err != nil {
        c.Error(err)
    }

    err = c.BindQuery(&ret.Order)
    if err != nil {
        c.Error(err)
    }

    err = c.BindQuery(&ret.Limit)
    if err != nil {
        c.Error(err)
    }

    if errScope.HasErrors() {
        return ret
    }

    {
        err := validation.ValidateStruct(&ret,
            validation.Field(
                &ret.Limit,
                validation.Max(100),
                validation.Min(20)),
            validation.Field(
                &ret.Order,
                validation.In(paginator.ASC, paginator.DESC)))
        if err != nil {
            c.Error(err)
        }
    }

    if ret.Limit == 0 {
        ret.Limit = 100
    }
    if ret.Order == "" {
        ret.Order = paginator.ASC
    }

    return ret
}

func CreatePaginator(c *gin.Context) paginator.Paginator {
    ret := paginator.Paginator{}

    errScope := CreateErrorScope(c)
    p := GetPagination(c)
    if errScope.HasErrors() {
        return ret
    }

    ret.SetKeys("ID")
    ret.SetOrder(p.Order)
    ret.SetLimit(p.Limit)

    {
        temp := p.Cursor.After
        if temp != nil {
            ret.SetAfterCursor(*temp)
        }
    }
    {
        temp := p.Cursor.Before
        if temp != nil {
            ret.SetBeforeCursor(*temp)
        }
    }
    return ret
}

func main() {
    var dbContext database_config.DatabasesContext
    {
        config_, err := config.ReadConfig();
        if err != nil {
            log.Fatal(err)
        }

        dbContext, err = database_config.EstablishConnectionsFromConfig(config_)
        if err != nil {
            log.Fatal(err)
        }
    }

    gormFactory := database.GormFactory{
        Config: gorm.Config{
            NamingStrategy: &database.MyNamer,
        },
        DbContext: &dbContext,
    }

    {
        gormDb, err := gormFactory.Create(dbContext.MainDatabaseIndex)
        _ = gormDb
        if err != nil {
            log.Fatal(err)
        }
    }

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

        errorCode := 400
        for _, err := range c.Errors {
            switch err.Err.(type) {
            case database.ConnectionError:
                errorCode = 500
            }
        }

        c.JSON(errorCode, gin.H{ "errors": errorStrings })
    })

    app.GET("/lists", func(c *gin.Context) {
        var query ListsQuery

        errScope := CreateErrorScope(c)

        var databaseIndex int
        if query.Database != "" {
            i_, found := dbContext.FindDatabaseWithName(query.Database)
            if !found {
                c.Error(fmt.Errorf("invalid database name: %s", query.Database))
            }
            databaseIndex = i_
        } else {
            databaseIndex = dbContext.MainDatabaseIndex
        }

        pagination := CreatePaginator(c)

        if errScope.HasErrors() {
            return
        }

        gormDb, err := gormFactory.Create(databaseIndex)
        if err != nil {
            log.Printf("Error while connecting to db: %v", err)
            c.Error(database.ConnectionError{})
            return
        }

        foiStatement := gormDb.Model(&database.Foaie{})
        var databaseFoi []database.Foaie
        gormDb, cursor, err := pagination.Paginate(foiStatement, &databaseFoi)
        if err != nil {
            log.Printf("Error while paginating: %v", err)
            c.Error(database.PaginationError{})
            return
        }

        // How to pass this back to the user with htmx??
        _ = cursor

        domainFoi := database.ToDomailModelsFoaie(databaseFoi)
        filteredLists := template_lists.FilteredLists{
            Values: domainFoi,
            FieldsShouldRender: foaie.AllFieldMask(),
            CurrencyFormatter: currency.ISO.Default(currency.EUR),
        }
        template := template_lists.Page(&filteredLists)
        renderTemplate(template, c)
    })

    source_map.Init(app, isDevelopment())

    app.Run()
}
