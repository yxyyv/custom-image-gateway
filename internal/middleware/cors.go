package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Cors() gin.HandlerFunc {

	return func(c *gin.Context) {

		var domain string
		if s, exist := c.GetQuery("domain"); exist {
			domain = s
		} else {
			domain = c.GetHeader("domain")
		}

		// 如果没有从 query 或 header 获取到 domain，尝试从反向代理头获取
		if domain == "" {
			// 优先使用 X-Forwarded-Host，这是反向代理传递的原始域名
			xForwardedHost := c.GetHeader("X-Forwarded-Host")
			if xForwardedHost != "" {
				domain = xForwardedHost
			} else if c.Request.Host != "" {
				// 回退到请求的 Host
				domain = c.Request.Host
			}
		}

		if domain != "" && !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			xForwardedProto := c.GetHeader("X-Forwarded-Proto")
			if xForwardedProto == "https" {
				domain = "https://" + domain
			} else {
				domain = "http://" + domain
			}
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With,  AccessToken, X-CSRF-Token, Authorization, Debug, Domain, Token, Lang, Content-Type, Content-Length,  Accept")

		if domain != "" {
			c.Header("Access-Control-Allow-Origin", domain)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// 允许放行OPTIONS请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}
