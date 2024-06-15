package stuff

import (
	"os"
	"strings"
)

func IsDevelopment() bool {
    env := os.Getenv("APP_ENV")
    isProd := strings.EqualFold(env, "PRODUCTION")
    return !isProd
}
