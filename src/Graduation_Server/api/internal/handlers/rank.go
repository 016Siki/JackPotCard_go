//ランク内のモード(ソロ、マルチ)の遊べるか取得API

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func RankModeHandler(w http.ResponseWriter, r *http.Request) {

	var canSolo, canMulti bool

	// ソロの可否確認
	err := db.QueryRow(`SELECT is_can_play FROM modes WHERE mode = 'ソロ'`).Scan(&canSolo)
	if err != nil {
		log.Printf("[ERROR] ソロモード取得失敗: %v", err)
		http.Error(w, "ソロモード取得失敗", http.StatusInternalServerError)
		return
	}

	// マルチの可否確認
	err = db.QueryRow(`SELECT is_can_play FROM modes WHERE mode = 'マルチ'`).Scan(&canMulti)
	if err != nil {
		log.Printf("[ERROR] マルチモード取得失敗: %v", err)
		http.Error(w, "マルチモード取得失敗", http.StatusInternalServerError)
		return
	}
	//レスポンス
	resp := map[string]interface{}{
		"result":    "OK",
		"can_solo":  canSolo,
		"can_multi": canMulti,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
