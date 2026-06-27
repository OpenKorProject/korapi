package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// Upstream holds the name and base URL of a backend service.
type Upstream struct {
	Name string
	URL  string
}

// Proxy routes incoming requests to registered upstream services.
type Proxy struct {
	upstreams map[string]*Upstream
	client    *http.Client
}

// NewProxy creates a new Proxy with the given HTTP client.
func NewProxy(client *http.Client) *Proxy {
	if client == nil {
		client = &http.Client{}
	}
	return &Proxy{
		upstreams: make(map[string]*Upstream),
		client:    client,
	}
}

// RegisterUpstream registers a named upstream service.
func (p *Proxy) RegisterUpstream(name, u string) {
	p.upstreams[name] = &Upstream{
		Name: name,
		URL:  strings.TrimRight(u, "/"),
	}
}

// Handler returns a Gin handler that proxies requests to the named upstream.
// pathPrefix is the gateway prefix (e.g. /v1/auth) that is preserved in the forwarded URL.
func (p *Proxy) Handler(pathPrefix, upstreamName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		upstream, ok := p.upstreams[upstreamName]
		if !ok {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":       "SERVICE_UNAVAILABLE",
					"message":    "Upstream service not configured",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		targetPath := c.Param("restOfPath")
		if targetPath == "" {
			targetPath = strings.TrimPrefix(c.Request.URL.Path, pathPrefix)
		}
		targetURL := fmt.Sprintf("%s%s%s", upstream.URL, pathPrefix, targetPath)

		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		u, err := url.Parse(targetURL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":       "BAD_REQUEST",
					"message":    "Invalid target URL",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		req := &http.Request{
			Method:        c.Request.Method,
			URL:           u,
			Header:        copyHeaders(c.Request.Header),
			Body:          c.Request.Body,
			ContentLength: c.Request.ContentLength,
		}

		// Forward gateway-injected headers to the backend
		if rid := c.GetString("request_id"); rid != "" {
			req.Header.Set("X-Request-ID", rid)
		}
		if uid := c.GetString("user_id"); uid != "" {
			req.Header.Set("X-User-ID", uid)
		}
		if tid := c.GetString("tenant_id"); tid != "" {
			req.Header.Set("X-Tenant-ID", tid)
		}
		if roles := c.GetString("roles"); roles != "" {
			req.Header.Set("X-Roles", roles)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":       "SERVICE_UNAVAILABLE",
					"message":    "Failed to reach upstream service",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			for _, vv := range v {
				c.Header(k, vv)
			}
		}
		c.Header("X-Request-ID", c.GetString("request_id"))

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":       "SERVICE_UNAVAILABLE",
					"message":    "Failed to read upstream response",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
}

// copyHeaders copies request headers, excluding hop-by-hop headers.
func copyHeaders(src http.Header) http.Header {
	dst := make(http.Header)
	for k, v := range src {
		if isHopByHop(k) {
			continue
		}
		dst[k] = append([]string{}, v...)
	}
	return dst
}

// isHopByHop reports whether a header is a hop-by-hop header per RFC 7230.
func isHopByHop(name string) bool {
	name = strings.ToLower(name)
	hopByHop := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailers":            true,
		"transfer-encoding":   true,
		"upgrade":             true,
	}
	return hopByHop[name]
}
