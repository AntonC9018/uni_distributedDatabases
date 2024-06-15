package stuff

import (
	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

func RenderTemplate(template templ.Component, c *gin.Context) {
    if err := template.Render(c.Request.Context(), c.Writer); err != nil {
        c.Error(err)
    }
}
