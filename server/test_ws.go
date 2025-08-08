package main

import (
	"encoding/json"
	"fmt"
	"log"
	"myflowhub/pkg/protocol"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func main() {
	// 连接到服务器
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()

	// 发送管理员认证
	authMsg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "manager_auth",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"token": "a-super-secret-manager-token",
		},
	}

	if err := conn.WriteJSON(authMsg); err != nil {
		log.Fatal("发送认证失败:", err)
	}

	// 读取认证响应
	var authResp protocol.BaseMessage
	if err := conn.ReadJSON(&authResp); err != nil {
		log.Fatal("读取认证响应失败:", err)
	}

	fmt.Printf("认证响应: %+v\n", authResp)

	// 发送节点查询
	nodesMsg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "query_nodes",
		Timestamp: time.Now(),
		Payload:   map[string]interface{}{},
	}

	if err := conn.WriteJSON(nodesMsg); err != nil {
		log.Fatal("发送节点查询失败:", err)
	}

	// 读取节点响应
	var nodesResp protocol.BaseMessage
	if err := conn.ReadJSON(&nodesResp); err != nil {
		log.Fatal("读取节点响应失败:", err)
	}

	fmt.Printf("节点响应类型: %s\n", nodesResp.Type)
	if payload, ok := nodesResp.Payload.(map[string]interface{}); ok {
		jsonData, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Printf("节点数据: %s\n", jsonData)
	}
	varsMsg := protocol.BaseMessage{
		ID:        uuid.New().String(),
		Type:      "query_variables",
		Timestamp: time.Now(),
		Payload: map[string]interface{}{
			"deviceId": "",
		},
	}

	if err := conn.WriteJSON(varsMsg); err != nil {
		log.Fatal("发送变量查询失败:", err)
	}

	// 读取变量响应
	var varsResp protocol.BaseMessage
	if err := conn.ReadJSON(&varsResp); err != nil {
		log.Fatal("读取变量响应失败:", err)
	}

	fmt.Printf("变量响应类型: %s\n", varsResp.Type)
	if payload, ok := varsResp.Payload.(map[string]interface{}); ok {
		jsonData, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Printf("变量数据: %s\n", jsonData)
	}
}
