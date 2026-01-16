# Go File Transfer (jp-file)

一个简单、高效的命令行文件传输CLI，支持跨平台（Windows, macOS, Linux）和广域网传输。

## 特性

*   **中继模式**：通过服务器中转，无需发送端和接收端在同一局域网。
*   **简单易用**：基于 4 位数字验证码进行配对传输。
*   **断点续传**：支持接收端持续运行，随时接收来自不同用户的文件。
*   **跨平台**：提供 Windows, macOS, Linux 的预编译二进制文件。
*   **自动配置**：自动记忆上次连接的服务器地址。

## 快速安装

使用以下命令一键安装最新版本（支持 macOS 和 Linux）：

```bash
curl -fsSL https://raw.githubusercontent.com/YOUR_USERNAME/go-file-transfer/main/install.sh | bash
```

或者手动下载 [Releases](https://github.com/YOUR_USERNAME/go-file-transfer/releases) 中的对应版本。

## 使用指南

### 1. 启动服务器 (jp-server)

你需要一台有公网 IP 的服务器。

```bash
./jp-server
# 默认监听 9999 端口
```

### 2. 接收文件 (Receiver)

在接收文件的电脑上运行：

```bash
# 首次运行需指定服务器地址
jp-file -s 192.168.0.250:9999

# 输出示例：
# Using Server: 192.168.0.250:9999
# Connected. Your code is: 3606
# Waiting for files... (Press Ctrl+C to exit)
```

程序会一直运行，等待接收文件。文件默认保存在当前目录，可通过 `-d` 指定目录：

```bash
jp-file -d ~/Downloads
```

**注意**：下次运行时，如果不指定 `-s`，程序会自动连接上次使用的服务器。

### 3. 发送文件 (Sender)

在发送文件的电脑上运行：

```bash
# 语法: jp-file <code> <filepath>
jp-file 3606 ./photo.jpg
```

**注意**：发送端也会自动使用上次配置的服务器地址。如果需要临时更改，可以加上 `-s` 参数。

## 编译指南

如果你想自己编译源码：

```bash
# 克隆仓库
git clone https://github.com/YOUR_USERNAME/go-file-transfer.git
cd go-file-transfer

# 编译
go mod tidy
./build.sh
```

## 协议说明

通信采用自定义 TCP 协议：
1.  **握手**：JSON 格式的消息交互（注册、连接、元数据）。
2.  **传输**：二进制流直传，由服务器中转。
