package source_map

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type ViteManifest struct {
    NameMap map[string]string
}

var viteManifest *ViteManifest

func initManifest(path string) {
    log.Printf("Reading %s", path)

    manifestBytes, err := os.ReadFile(path)
    if err != nil {
        panic(err)
    }

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		panic(err)
	}

	result := &ViteManifest{
		NameMap: make(map[string]string),
	}

	for key, value := range manifest {
		entry, ok := value.(map[string]interface{})
		if !ok {
			panic("unexpected manifest format")
		}

        if !strings.HasPrefix(key, "static/") {
            panic("expected file to be in `static/`")
        }
        key = key[len("static/"):]

		file, ok := entry["file"].(string)
		if !ok {
			panic("unexpected file format")
		}
		result.NameMap[key] = file
	}

    viteManifest = result
}

func InitSourceMapping(engine *gin.Engine, isDevelopment bool) {

    if !isDevelopment {
        initManifest("dist/manifest.json")
        engine.Static("dist", "dist")
        return
    }

    vitePort := "5173"
    remote, err := url.Parse("http://localhost:" + vitePort)
    if err != nil {
        panic(err)
    }

    // Reverse proxy
    engine.Any("/dist/*catchall", func(c *gin.Context) {
        proxy := httputil.NewSingleHostReverseProxy(remote)
        proxy.Director = func(req *http.Request) {
            req.Host = remote.Host
            req.Header = c.Request.Header
            req.URL = remote
            req.URL.Path = c.Param("catchall")
        }
        proxy.ServeHTTP(c.Writer, c.Request)
    })
}

func RemapSource(path string) string {
    if viteManifest != nil {
        return viteManifest.NameMap[path]
    }
    // This only happens in development, so we don't care about the allocation
    return "dist/" + path
}
