package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type PlayerMessage struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Chips      int    `json:"chips"`
	Action     string `json:"action"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Unityからの接続を許可
	},
}

func BlackjackWebSocketHandle(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("✅ WebSocket 接続成功")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("接続終了:", err)
			break
		}

		var msg PlayerMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("JSON解析エラー:", err)
			continue
		}

		log.Printf("📩 受信: プレイヤーID:%s 名前:%s チップ:%d アクション:%s\n",
			msg.PlayerID, msg.PlayerName, msg.Chips, msg.Action)

		// 今は受け取ったメッセージをそのまま返す（エコーバック）
		if err := conn.WriteJSON(msg); err != nil {
			log.Println("書き込みエラー:", err)
			break
		}
	}
}
