package stuff

import (
	"github.com/gin-gonic/gin"
)

type CookieConfig struct {
    Name string
    Age int
}

func SetCookie(c *gin.Context, config *CookieConfig, value string) {
    httpOnly := true
    secure := true
    c.SetCookie(
        config.Name,
        value,
        config.Age,
        "/",
        "localhost",
        secure,
        httpOnly)
}

func RemoveCookie(c *gin.Context, config *CookieConfig) {
    _, err := c.Cookie(config.Name)
    noCookie := err != nil
    if noCookie {
        return
    }
    
    copy := *config
    copy.Age = -1
    SetCookie(c, &copy, "")
}

func GetCookie(c *gin.Context, config *CookieConfig) (string, error) {
    return c.Cookie(config.Name)
}
