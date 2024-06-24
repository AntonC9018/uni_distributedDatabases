package clients

import (
	"webapp/features/clients/create"
	"webapp/stuff"

	"github.com/gin-gonic/gin"
)

func InitHandlers(builder *gin.Engine, appContext *stuff.ApplicationContext) {
    create.InitHandler(builder, appContext)
}
