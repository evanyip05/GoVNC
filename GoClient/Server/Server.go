package Server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Serve() {
	router := gin.Default()

	// allow cors
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// serve frontend
	router.StaticFile("/manifest.json", "Frontend/build/manifest.json")
	router.StaticFile("/", "Frontend/build/index.html")
	router.StaticFS("/public", http.Dir(`Frontend/public`))
	router.Static("/static", "Frontend/build/static")

	err := router.Run(":8080")
	if err != nil {
		println("Error serving FE", err.Error())
	}
}
