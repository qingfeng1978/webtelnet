# WebTelnet项目部署报告 - 2025-03-22

## 部署方案概览

本项目采用域名+端口的直接访问方式部署：

- 前端服务: 域名:8080
- 后端API服务: 域名:3000
- 数据库服务: 本地端口3306
- Telnet服务器: 标准23端口(内部服务)

## 一、准备工作

1. **服务器准备**
   - Linux服务器(Ubuntu 20.04/CentOS 8)
   - 至少2GB内存
   - 开放端口: 8080和3000

2. **安装基础软件**
   ```bash
   # 安装软件包
   sudo apt update
   sudo apt install -y git nodejs npm mariadb-server telnetd
   
   # 安装PM2
   sudo npm install -g pm2
   ```

## 二、数据库配置

1. **启动数据库服务**
   ```bash
   sudo systemctl start mariadb
   sudo systemctl enable mariadb
   sudo mysql_secure_installation
   ```

2. **创建数据库和用户**
   ```bash
   sudo mysql -u root -p
   CREATE DATABASE webtelnet;
   CREATE USER 'webtelnet_user'@'localhost' IDENTIFIED BY '密码';
   GRANT ALL PRIVILEGES ON webtelnet.* TO 'webtelnet_user'@'localhost';
   FLUSH PRIVILEGES;
   EXIT;
   ```

## 三、Telnet服务器配置

1. **启用Telnet服务**
   ```bash
   sudo systemctl start telnet.socket
   sudo systemctl enable telnet.socket
   ```

2. **限制Telnet安全访问**
   ```bash
   # 编辑hosts.allow和hosts.deny
   sudo nano /etc/hosts.allow
   # 添加: in.telnetd: 127.0.0.1
   
   sudo nano /etc/hosts.deny
   # 添加: in.telnetd: ALL
   ```

## 四、前端部署

1. **构建前端**
   ```bash
   cd frontend
   npm install
   npm run build
   ```

2. **启动前端服务**
   ```bash
   npm install -g serve
   pm2 start "serve -s dist -l 8080" --name webtelnet-frontend
   ```

## 五、后端部署

1. **配置后端**
   ```bash
   cd backend
   npm install
   
   # 创建.env文件
   echo "PORT=3000
   DB_HOST=localhost
   DB_USER=webtelnet_user
   DB_PASSWORD=密码
   DB_NAME=webtelnet
   TELNET_HOST=localhost
   TELNET_PORT=23" > .env
   ```

2. **启动后端服务**
   ```bash
   pm2 start app.js --name webtelnet-backend
   pm2 startup
   pm2 save
   ```

## 六、防火墙配置

1. **开放必要端口**
   ```bash
   sudo ufw allow 8080/tcp
   sudo ufw allow 3000/tcp
   ```

## 七、验证部署

1. **测试服务**
   - 前端: http://域名:8080
   - 后端: http://域名:3000/api/status
   - 数据库: mysql -u webtelnet_user -p
   - Telnet: telnet localhost

## 部署流程图

用户 → 前端Web界面(:8080) → 后端API(:3000) → Telnet服务器(:23) → 远程服务器
                                    ↕
                               数据库(:3306)

## 维护提示

1. **查看日志**
   ```bash
   pm2 logs
   ```

2. **备份数据库**
   ```bash
   mysqldump -u webtelnet_user -p webtelnet > backup_$(date +%Y%m%d).sql
   ```

部署日期: 2025-03-22
