# 1. Chat Demo

- [1. Chat Demo](#1-chat-demo)
  - [1.1. 环境准备](#11-环境准备)
    - [1.1.1. 安装Golang](#111-安装golang)
    - [1.1.2. 代理设置](#112-代理设置)
    - [1.1.3. 安装VSCODE](#113-安装vscode)
  - [1.2. 启动服务](#12-启动服务)
  - [1.3. 测试](#13-测试)
  - [1.4. Java to Go](#14-java-to-go)

> 这是一个简单的websocket协议demo。

## 1.1. 环境准备

### 1.1.1. 安装Golang

打开 https://golang.google.cn/dl/ ，下载对应的安装包。

**在Mac下可以在终端执行**:

>brew install golang

### 1.1.2. 代理设置

打开你的终端并执行

```cmd
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct
```

> https://goproxy.cn/

### 1.1.3. 安装VSCODE

从 https://code.visualstudio.com/ 下载`Visual Studio Code`。

安装完vscode之后，就可以打开本demo项目了，它会提示你安装一些`扩展和Golang的环境工具`，确认安装即可。

## 1.2. 启动服务

> go run main.go chat

```shell
$ go run main.go chat
INFO[0000] started                                       id=demo listen=":8000" module=Server
```

## 1.3. 测试

打开两个websocket测试界面中，并分别输入：

```html
ws://localhost:8000?user=userA
ws://localhost:8000?user=userB
```

之后就可以通过界面发送聊天消息了

## 1.4. Java to Go

- [Java to Go in-depth tutorial](https://yourbasic.org/golang/go-java-tutorial/)
