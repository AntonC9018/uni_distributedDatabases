package clients

import (
	"common/models"
	"webapp/features/clients/create"
	"webapp/features/clients/create/templates"
	"webapp/stuff"

	"github.com/gin-gonic/gin"
)

func InitHandlers(builder *gin.Engine, appContext *stuff.ApplicationContext) {
    create.InitHandler(builder, appContext)

    builder.GET("client/debug", func(c *gin.Context) {
        client := models.Client{
            ID: 1,
            Email: "hello",
        }
        params := templates.DebugInfoParams{
            Client: &client,
        }
        template := templates.DebugInfo(params)
        stuff.RenderTemplate(template, c)
    })
}
