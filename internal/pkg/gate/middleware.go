package gate

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin middleware that enforces gate authentication.
//
// loginPath is the URL of the gate login page (e.g. "/gate" or "/arcon/gate").
// If the request carries a valid session cookie, the request proceeds normally.
// Otherwise the client is redirected to loginPath with the original request URI
// preserved as the ?redirect= query parameter so the user lands at their
// intended destination after a successful login.
func Middleware(svc *Service, loginPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc.IsAuthenticated(c) {
			c.Next()
			return
		}

		// Build the login URL, preserving the original destination.
		target := loginPath
		if orig := c.Request.URL.RequestURI(); orig != "" && orig != "/" {
			target = loginPath + "?redirect=" + url.QueryEscape(orig)
		}

		c.Redirect(http.StatusFound, target)
		c.Abort()
	}
}
