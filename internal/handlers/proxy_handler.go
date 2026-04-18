package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// GET /api/v1/proxy/image?url=<encoded_url>
// Proxy external images to avoid CORS issues in the browser.
func ImageProxy(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.Status(http.StatusBadRequest)
		return
	}

	resp, err := http.Get(rawURL) //nolint:noctx
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}

	c.Header("Cache-Control", "public, max-age=86400")
	c.DataFromReader(resp.StatusCode, resp.ContentLength, contentType, resp.Body, nil)
}
