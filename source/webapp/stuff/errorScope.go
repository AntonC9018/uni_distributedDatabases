package stuff

import "github.com/gin-gonic/gin"

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
