package stuff

import (
	"webapp/database"

	"github.com/gin-gonic/gin"
)

func InitErrorHandler(app *gin.Engine) {
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
}

type ValidationError error

