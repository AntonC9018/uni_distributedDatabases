package lists

import (
	database_config "common/database"
	"common/models/foaie"
	"fmt"
	"log"
	"webapp/database"
	"webapp/stuff"
    "webapp/features/lists/templates"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/currency"
)

type ListsQuery struct {
    Database string `query:"db"`
}

type handleListParams struct {
    DbContext database_config.DatabasesContext
    GormFactory database.GormFactory
    Context *gin.Context
}

func getListsTemplateData(p handleListParams) (ret templates.FilteredLists) {
    var query ListsQuery

    errScope := stuff.CreateErrorScope(p.Context)

    if err := p.Context.Bind(&query); err != nil {
        p.Context.Error(err)
    }
    pagination := stuff.CreatePaginator(p.Context)

    if errScope.HasErrors() {
        return
    }

    var databaseIndex int
    if query.Database != "" {
        i_, found := p.DbContext.FindDatabaseWithName(query.Database)
        if !found {
            p.Context.Error(fmt.Errorf("invalid database name: %s", query.Database))
        }
        databaseIndex = i_
    } else {
        databaseIndex = p.DbContext.MainDatabaseIndex
    }

    if errScope.HasErrors() {
        return
    }

    gormDb, err := p.GormFactory.Create(databaseIndex)
    if err != nil {
        log.Printf("Error while connecting to db: %v", err)
        p.Context.Error(database.ConnectionError{})
        return
    }

    foiStatement := gormDb.Model(&database.Foaie{})
    var databaseFoi []database.Foaie
    gormDb, cursor, err := pagination.Paginate(foiStatement, &databaseFoi)
    if err != nil {
        log.Printf("Error while paginating: %v", err)
        p.Context.Error(database.PaginationError{})
        return
    }

    urlString := func() string {
        if cursor.After == nil {
            return ""
        }
        urlCopy := *p.Context.Request.URL
        urlCopy.Path = itemsPath
        stuff.ReplaceCursorInQuery(&urlCopy, cursor)
        return urlCopy.String()
    }()

    domainFoi := database.ToDomailModelsFoaie(databaseFoi)
    ret = templates.FilteredLists{
        Values: domainFoi,
        FieldsShouldRender: foaie.AllFieldMask(),
        CurrencyFormatter: currency.ISO.Default(currency.EUR),
        NextItemsUrl: urlString,
    }
    return
}

type TemplateType int
const (
    Page TemplateType = iota
    Items = iota
)

const TemplateCount = 2
const itemsPath = "list-items"

func renderListsTemplate(templateType TemplateType, p handleListParams) {

    errScope := stuff.CreateErrorScope(p.Context)

    data := getListsTemplateData(p)
    if errScope.HasErrors() {
        return
    }

    template := func() templ.Component {
        switch templateType {
        case Page:
            return templates.Page(&data)
        case Items:
            return templates.Lists(&data)
        default:
            panic("Unreachable")
        }
    }()

    stuff.RenderTemplate(template, p.Context)
}

func InitListHandler(builder *gin.Engine, appContext *stuff.ApplicationContext) {
    for i := 0; i < TemplateCount; i++ {
        templateType := TemplateType(i)
        path := func() string {
            switch templateType {
            case Page:
                return "/lists"
            case Items:
                return itemsPath
            default:
                panic("unreachable")
            }
        }()
        builder.GET(path, func(c *gin.Context) {
            params := handleListParams{
                DbContext: appContext.DbContext,
                GormFactory: appContext.GormFactory,
                Context: c,
            }
            renderListsTemplate(templateType, params)
        })
    }
}
