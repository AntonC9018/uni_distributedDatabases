package create

import (
	"webapp/features/clients/create/createhandler"
	"webapp/features/clients/create/formhandler"
	"webapp/features/clients/create/templates"
	"webapp/stuff"

	"github.com/gin-gonic/gin"
)

func InitHandler(builder *gin.Engine, appContext *stuff.ApplicationContext) {
    g := builder.Group("client")

    g.POST("", func(c *gin.Context) {
        createhandler.Handle(c, appContext)

        params := templates.CreateResultParams{
            Errors: c.Errors,
        }
        template := templates.CreateResult(&params)
        err := template.Render(c.Request.Context(), c.Writer);
        c.Errors = c.Errors[:0]
        if err != nil {
            c.Error(err)
        }
    })

    g.GET("form", func(c *gin.Context) {
        errScope := stuff.CreateErrorScope(c)
        client := formhandler.Handle(c, appContext)
        if errScope.HasErrors() {
            return
        }

        params := templates.CreateFormParams{
            Client: &client,
        }
        template := templates.CreateFormPage(&params)
        stuff.RenderTemplate(template, c)
    })
}
