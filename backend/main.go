package main

import (
	"fmt"
	"log"
	"webtelnet/api"
	"webtelnet/config"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置
	if err := config.Init(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// 创建Gin路由
	r := gin.Default()

	// 配置CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// WebSocket路由
	r.GET("/ws", api.HandleWebSocket)
	r.GET("/ws/:sessionID", api.HandleWebSocket)

	// 服务器管理API
	r.GET("/servers", api.GetServers)
	r.GET("/servers/tree", api.GetServerTree)
	r.GET("/servers/:id", api.GetServer)
	r.POST("/servers", api.CreateServer)
	r.DELETE("/servers/:id", api.DeleteServer)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", config.GlobalConfig.Server.Host, config.GlobalConfig.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
