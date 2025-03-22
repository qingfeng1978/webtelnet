package models

type Server struct {
	ID        int    `json:"id"`
	ParentID  int    `json:"parent_id"`
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
	CreatedAt string `json:"created_at"`
}

// ServerTree 用于构建树形结构
type ServerTree struct {
	Server   Server       `json:"server"`
	Children []ServerTree `json:"children"`
} 