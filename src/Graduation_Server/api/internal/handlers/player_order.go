package handlers

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

// --- player_order メッセージ構造体 ---
type PlayerOrderMessage struct {
	Type    string       `json:"type"` // "player_order"
	Players []PlayerInfo `json:"players"`
}

// --- プレイヤー順をWebSocketへ送信 ---
func SendPlayerOrder(conn *websocket.Conn, players []PlayerInfo) error {

	msg := PlayerOrderMessage{
		Type:    "player_order",
		Players: players,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}
