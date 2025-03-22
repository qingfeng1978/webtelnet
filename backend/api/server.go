package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"webtelnet/models"
)

// 模拟数据库 - 在实际应用中应该使用数据库
var servers = []models.Server{
	{ID: 1, ParentID: 0, Name: "服务器组1", Host: "", Port: 0},
	{ID: 2, ParentID: 1, Name: "本地Telnet", Host: "localhost", Port: 23},
	{ID: 3, ParentID: 0, Name: "服务器组2", Host: "", Port: 0},
	{ID: 4, ParentID: 3, Name: "测试服务器", Host: "example.com", Port: 23},
}

// GetServers 获取所有服务器列表
func GetServers(c *gin.Context) {
	c.JSON(http.StatusOK, servers)
}

// GetServerTree 获取树形结构的服务器列表
func GetServerTree(c *gin.Context) {
	// 构建树形结构
	var result []models.ServerTree
	buildServerTree(&result, 0)
	c.JSON(http.StatusOK, result)
}

// 递归构建树形结构
func buildServerTree(trees *[]models.ServerTree, parentID int) {
	for _, server := range servers {
		if server.ParentID == parentID {
			tree := models.ServerTree{
				Server:   server,
				Children: []models.ServerTree{},
			}
			// 递归查找子节点
			buildServerTree(&tree.Children, server.ID)
			*trees = append(*trees, tree)
		}
	}
}

// GetServer 获取单个服务器信息
func GetServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	for _, server := range servers {
		if server.ID == id {
			c.JSON(http.StatusOK, server)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
}

// CreateServer 创建服务器
func CreateServer(c *gin.Context) {
	var server models.Server
	if err := c.ShouldBindJSON(&server); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置新ID (实际应用中应该由数据库自动生成)
	server.ID = len(servers) + 1
	servers = append(servers, server)

	c.JSON(http.StatusCreated, server)
}

// DeleteServer 删除服务器
func DeleteServer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	for i, server := range servers {
		if server.ID == id {
			// 删除该服务器
			servers = append(servers[:i], servers[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "服务器不存在"})
} 