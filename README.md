# chatdemo

这是一个websocket最简单的demo项

## 启动服务

> go run main.go chat

```shell
$ go run main.go chat
INFO[0000] started                                       id=demo listen=":8000" module=Server
```

## 测试

打开两个websocket测试界面中，并分别输入：

```html
ws://localhost:8000?user=userA
ws://localhost:8000?user=userB
```

之后就可以通过界面发送聊天消息了
