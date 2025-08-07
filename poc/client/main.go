package main

import (
	"log"
	"net/url"
	"time"

	"myflowhub/poc/protocol"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	log.Printf("正在连接到 %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer c.Close()

	log.Println("已成功连接到中枢")

	// 创建一个符合新协议的认证请求消息
	authPayload := protocol.AuthRequestPayload{
		DeviceID:  "node-001",
		SecretKey: "super-secret",
	}
	message := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Target:    "server",
		Type:      "auth_request",
		Timestamp: time.Now(),
		Payload:   authPayload,
	}

	// 发送 JSON 消息
	log.Println("正在发送认证请求...")
	err = c.WriteJSON(message)
	if err != nil {
		log.Println("写入JSON错误:", err)
		return
	}

	// 读取并解析响应
	var response protocol.BaseMessage
	err = c.ReadJSON(&response)
	if err != nil {
		log.Println("读取JSON错误:", err)
		return
	}
	log.Printf("收到响应: %+v\n", response)

	time.Sleep(1 * time.Second)
	log.Println("关闭连接")
}
