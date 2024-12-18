package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/xendit/xendit-go/v6"
)

func XenditMiddleware(xenditClient *xendit.APIClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("xendit_client", xenditClient)
		c.Next()
	}
}
func GetXenditClient(c *gin.Context) *xendit.APIClient {
	client, exists := c.Get("xendit_client")
	if !exists {
		return nil
	}
	return client.(*xendit.APIClient)
}
