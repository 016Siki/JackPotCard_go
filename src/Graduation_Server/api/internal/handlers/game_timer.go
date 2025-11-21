package handlers

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type TimerMessage struct {
	Type      string  `json:"type"`      // "timer"
	Remaining float64 `json:"remaining"` // 残り秒（0.1秒単位）
}

// 0.1秒間隔（100ms）で送信するタイマー
func startActionTimer(conn *websocket.Conn, totalSec float64) {
	go func() {
		remaining := totalSec

		// 0.1秒(100ms)ごとの ticker
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			// 送信メッセージ
			msg := TimerMessage{
				Type:      "timer",
				Remaining: remaining,
			}

			// WebSocket送信
			if err := writeJSONSafe(conn, msg); err != nil {
				log.Println("timer send error:", err)
				return
			}

			log.Printf("timer: remaining=%.1f", remaining)

			// 終了
			if remaining <= 0 {
				log.Println("timer finished")
				return
			}

			// 0.1秒減算
			remaining = remaining - 0.1

			// 100ms待機
			<-ticker.C
		}
	}()
}
