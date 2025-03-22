package api

import (
	"fmt"
	"io"
	"golang.org/x/crypto/ssh"
	"time"
)

// SSHConfig 存储SSH连接配置
type SSHConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateSSHClient 创建SSH客户端连接
func CreateSSHClient(config *SSHConfig) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 在生产环境中应该验证主机密钥
		Timeout:        15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	return ssh.Dial("tcp", addr, sshConfig)
}

// CreateSSHSession 创建SSH会话
func CreateSSHSession(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	// 请求伪终端
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		session.Close()
		return nil, err
	}

	return session, nil
}

// HandleSSHSession 处理SSH会话的输入输出
func HandleSSHSession(session *ssh.Session, input io.Reader, output io.Writer) error {
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	// 复制输入到SSH会话
	go func() {
		io.Copy(stdin, input)
	}()

	// 复制SSH会话输出
	go func() {
		io.Copy(output, stdout)
	}()

	// 复制错误输出
	go func() {
		io.Copy(output, stderr)
	}()

	// 启动远程shell
	return session.Shell()
} 