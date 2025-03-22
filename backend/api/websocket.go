package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境下允许所有跨域请求
	},
}

type Connection struct {
	WSConn     *websocket.Conn
	NetConn    net.Conn
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	LastActive time.Time
	Mutex      sync.Mutex
	Protocol   string // "telnet" or "ssh"
}

// ConnectionRequest 连接请求结构
type ConnectionRequest struct {
	Protocol string     `json:"protocol"` // "telnet" or "ssh"
	Host     string     `json:"host"`
	Port     int        `json:"port"`
	SSH      *SSHConfig `json:"ssh,omitempty"`
}

// 活动连接池
var connectionPool = struct {
	sync.Mutex
	conns map[string]*Connection
}{
	conns: make(map[string]*Connection),
}

// 添加连接
func addConnection(id string, conn *Connection) {
	connectionPool.Lock()
	defer connectionPool.Unlock()
	connectionPool.conns[id] = conn
	// 打印当前连接数
	log.Printf("Connection added: %s, total connections: %d", id, len(connectionPool.conns))
}

// 删除连接
func removeConnection(id string) {
	connectionPool.Lock()
	defer connectionPool.Unlock()
	delete(connectionPool.conns, id)
	log.Printf("Connection removed: %s, remaining connections: %d", id, len(connectionPool.conns))
}

// 更新心跳时间
func updateHeartbeat(id string) {
	connectionPool.Lock()
	defer connectionPool.Unlock()
	if conn, ok := connectionPool.conns[id]; ok {
		conn.Mutex.Lock()
		conn.LastActive = time.Now()
		conn.Mutex.Unlock()
	}
}

// 检查连接是否超时
func checkConnectionTimeout() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		connectionPool.Lock()
		timeoutThreshold := time.Now().Add(-2 * time.Minute)
		var expiredIDs []string

		for id, conn := range connectionPool.conns {
			conn.Mutex.Lock()
			if conn.LastActive.Before(timeoutThreshold) {
				expiredIDs = append(expiredIDs, id)
			}
			conn.Mutex.Unlock()
		}
		connectionPool.Unlock()

		// 清理过期连接
		for _, id := range expiredIDs {
			connectionPool.Lock()
			if conn, ok := connectionPool.conns[id]; ok {
				conn.WSConn.Close()
				if conn.NetConn != nil {
					conn.NetConn.Close()
				}
				if conn.SSHSession != nil {
					conn.SSHSession.Close()
				}
				if conn.SSHClient != nil {
					conn.SSHClient.Close()
				}
				delete(connectionPool.conns, id)
				log.Printf("Connection timeout: %s", id)
			}
			connectionPool.Unlock()
		}
	}
}

// 初始化连接管理
func init() {
	go checkConnectionTimeout()
}

func HandleWebSocket(c *gin.Context) {
	// 从URL路径参数中获取会话ID
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		// 如果未提供会话ID，则生成一个
		sessionID = fmt.Sprintf("auto_%s_%s", c.ClientIP(), time.Now().Format("20060102150405"))
	}

	log.Printf("处理WebSocket连接请求，会话ID: %s", sessionID)

	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("升级连接失败: %v", err)
		return
	}

	// 生成唯一连接ID，包含会话ID
	connID := fmt.Sprintf("%s_%s", sessionID, time.Now().Format("20060102150405"))

	// 设置WebSocket读取超时
	wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// 等待接收连接信息
	_, message, err := wsConn.ReadMessage()
	if err != nil {
		log.Printf("读取连接信息失败: %v", err)
		wsConn.Close()
		return
	}

	// 解析连接请求
	var req ConnectionRequest
	if err := json.Unmarshal(message, &req); err != nil {
		log.Printf("解析连接请求失败: %v", err)
		wsConn.WriteMessage(websocket.TextMessage, []byte("无效的连接请求"))
		wsConn.Close()
		return
	}

	// 查找是否已有相同会话ID的连接
	connectionPool.Lock()
	var existingConn *Connection
	var existingConnID string

	for id, conn := range connectionPool.conns {
		if strings.HasPrefix(id, sessionID+"_") {
			existingConn = conn
			existingConnID = id
			break
		}
	}
	connectionPool.Unlock()

	// 如果找到相同会话的连接，则关闭旧连接
	if existingConn != nil {
		log.Printf("找到同一会话ID的现有连接: %s，准备替换", existingConnID)
		existingConn.WSConn.Close()
		if existingConn.NetConn != nil {
			existingConn.NetConn.Close()
		}
		if existingConn.SSHSession != nil {
			existingConn.SSHSession.Close()
		}
		if existingConn.SSHClient != nil {
			existingConn.SSHClient.Close()
		}
		removeConnection(existingConnID)
	}

	// 创建连接对象
	conn := &Connection{
		WSConn:     wsConn,
		LastActive: time.Now(),
		Protocol:   req.Protocol,
	}

	// 根据协议类型建立连接
	switch req.Protocol {
	case "telnet":
		targetAddr := net.JoinHostPort(req.Host, strconv.Itoa(int(req.Port)))
		telnetConn, err := net.Dial("tcp", targetAddr)
		if err != nil {
			log.Printf("连接Telnet服务器失败: %v", err)
			wsConn.WriteMessage(websocket.TextMessage, []byte("连接失败"))
			wsConn.Close()
			return
		}
		conn.NetConn = telnetConn

	case "ssh":
		if req.SSH == nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte("需要SSH配置"))
			wsConn.Close()
			return
		}

		sshClient, err := CreateSSHClient(req.SSH)
		if err != nil {
			log.Printf("创建SSH客户端失败: %v", err)
			wsConn.WriteMessage(websocket.TextMessage, []byte("SSH连接失败"))
			wsConn.Close()
			return
		}

		sshSession, err := CreateSSHSession(sshClient)
		if err != nil {
			log.Printf("创建SSH会话失败: %v", err)
			sshClient.Close()
			wsConn.WriteMessage(websocket.TextMessage, []byte("创建SSH会话失败"))
			wsConn.Close()
			return
		}

		conn.SSHClient = sshClient
		conn.SSHSession = sshSession

	default:
		wsConn.WriteMessage(websocket.TextMessage, []byte("不支持的协议"))
		wsConn.Close()
		return
	}

	// 添加到连接池
	addConnection(connID, conn)

	log.Printf("成功建立连接: %s (会话ID: %s, 协议: %s)", connID, sessionID, req.Protocol)

	// 确保连接资源被释放
	defer func() {
		log.Printf("正在释放连接: %s (会话ID: %s)", connID, sessionID)
		wsConn.Close()
		if conn.NetConn != nil {
			conn.NetConn.Close()
		}
		if conn.SSHSession != nil {
			conn.SSHSession.Close()
		}
		if conn.SSHClient != nil {
			conn.SSHClient.Close()
		}
		removeConnection(connID)
	}()

	// 重置WebSocket读取超时
	wsConn.SetReadDeadline(time.Time{})

	// 创建双向数据通道
	done := make(chan bool)

	if conn.Protocol == "telnet" {
		// Telnet数据处理
		handleTelnetConnection(conn, done, connID)
	} else {
		// SSH数据处理
		handleSSHConnection(conn, done, connID)
	}

	<-done // 等待任一方连接断开
}

// handleTelnetConnection 处理Telnet连接的数据传输
func handleTelnetConnection(conn *Connection, done chan bool, connID string) {
	// Telnet -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		var logoutBuffer string
		for {
			n, err := conn.NetConn.Read(buf)
			if err != nil {
				log.Printf("Failed to read from telnet: %v", err)
				// 检查是否是正常关闭
				if err == io.EOF || strings.Contains(logoutBuffer, "Configuration console exit") {
					// 发送正常关闭的消息
					conn.WSConn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, "manual_exit"))
				} else {
					// 发送异常关闭的消息
					conn.WSConn.WriteMessage(websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error()))
				}
				done <- true
				return
			}

			// 更新退出检测缓冲区
			newText := string(buf[:n])
			logoutBuffer += newText
			if len(logoutBuffer) > 1024 { // 保持缓冲区大小合理
				logoutBuffer = logoutBuffer[len(logoutBuffer)-1024:]
			}

			// 检查是否包含退出确认信息
			if strings.Contains(logoutBuffer, "Are you sure to log out? (y/n)[n]:y") ||
				strings.Contains(logoutBuffer, "Configuration console exit") {
				// 发送正常关闭消息
				conn.WSConn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "manual_exit"))
				done <- true
				return
			}

			err = conn.WSConn.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				log.Printf("Failed to write to websocket: %v", err)
				done <- true
				return
			}
			// 更新心跳时间
			updateHeartbeat(connID)
		}
	}()

	// WebSocket -> Telnet
	go func() {
		for {
			_, message, err := conn.WSConn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Printf("WebSocket closed normally")
				} else {
					log.Printf("Failed to read from websocket: %v", err)
				}
				done <- true
				return
			}

			// 检查是否是心跳消息
			if len(message) == 1 && message[0] == 0 {
				updateHeartbeat(connID)
				continue
			}

			_, err = conn.NetConn.Write(message)
			if err != nil {
				log.Printf("Failed to write to telnet: %v", err)
				done <- true
				return
			}
			// 更新心跳时间
			updateHeartbeat(connID)
		}
	}()
}

// handleSSHConnection 处理SSH连接的数据传输
func handleSSHConnection(conn *Connection, done chan bool, connID string) {
	// 创建管道用于数据传输
	wsReader, wsWriter := io.Pipe()
	sshReader, sshWriter := io.Pipe()

	// 处理SSH会话
	go func() {
		if err := HandleSSHSession(conn.SSHSession, wsReader, sshWriter); err != nil {
			log.Printf("SSH session error: %v", err)
			done <- true
		}
	}()

	// WebSocket -> SSH
	go func() {
		for {
			_, data, err := conn.WSConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error: %v", err)
				}
				done <- true
				return
			}

			if len(data) == 1 && data[0] == 0 {
				updateHeartbeat(connID)
				continue
			}

			_, err = wsWriter.Write(data)
			if err != nil {
				log.Printf("Failed to write to SSH: %v", err)
				done <- true
				return
			}

			updateHeartbeat(connID)
		}
	}()

	// SSH -> WebSocket
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := sshReader.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Failed to read from SSH: %v", err)
				}
				done <- true
				return
			}

			conn.Mutex.Lock()
			err = conn.WSConn.WriteMessage(websocket.BinaryMessage, buf[:n])
			conn.Mutex.Unlock()

			if err != nil {
				log.Printf("Failed to write to websocket: %v", err)
				done <- true
				return
			}

			updateHeartbeat(connID)
		}
	}()
}
