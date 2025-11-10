// ランクモード、フレンドモードが遊べるかを確認API
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func GameModeHandler(w http.ResponseWriter, r *http.Request) {
	//遊べるゲームの確認
	var canRank, canFriend bool
	// ランクの可否確認
	err := db.QueryRow(`SELECT is_can_play FROM modes WHERE mode = 'ランク'`).Scan(&canRank)
	if err != nil {
		log.Printf("[ERROR] ランクモード取得失敗: %v", err)
		http.Error(w, "ランクモード取得失敗", http.StatusInternalServerError)
		return
	}

	// フレンドの可否確認
	err = db.QueryRow(`SELECT is_can_play FROM modes WHERE mode = 'フレンド'`).Scan(&canFriend)
	if err != nil {
		log.Printf("[ERROR] フレンドモード取得失敗: %v", err)
		http.Error(w, "フレンドモード取得失敗", http.StatusInternalServerError)
		return
	}
	//レスポンスデータ
	resp := map[string]interface{}{
		"result":     "OK",
		"can_rank":   canRank,
		"can_friend": canFriend,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

}
