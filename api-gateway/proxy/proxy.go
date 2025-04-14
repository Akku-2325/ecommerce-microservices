package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewProxyHandler(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Invalid target URL for proxy: %s, error: %v", targetURL, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", target.Host)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		apiBasePath := "/api/v1"
		if strings.HasPrefix(req.URL.Path, apiBasePath) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, apiBasePath)
			log.Printf("Rewriting path from %s to %s", apiBasePath+req.URL.Path, req.URL.Path)
		}

		req.Host = target.Host
		log.Printf("Proxying request: %s %s%s to %s", req.Method, req.Host, req.URL.Path, targetURL)
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("Proxy error: %v", err)
		http.Error(rw, "Proxy Error", http.StatusBadGateway)
	}

	return func(c *gin.Context) {
		log.Printf("Incoming request to gateway: %s %s", c.Request.Method, c.Request.URL.Path)
		proxy.ServeHTTP(c.Writer, c.Request)
		log.Printf("Finished proxying request for: %s %s | Status: %d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
	}
}
