//マルチゲーム判定API

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func FriendGameListHandler(w http.ResponseWriter, r *http.Request) {
	//DBのマルチゲームでtrueのものを取得する
	query := `
	SELECT t.id, g.code, t.name, t.rule
	FROM types t
	JOIN game_types g ON t.game_type_id = g.id
	WHERE t.mode_id = (SELECT id FROM modes WHERE mode = 'フレンド') AND t.is_can_play = TRUE
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[ERROR] multi select failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []map[string]interface{}
	for rows.Next() {
		var id int
		var typeCode, name, rule string
		if err := rows.Scan(&id, &typeCode, &name, &rule); err != nil {
			log.Printf("[ERROR] friend scan failed: %v", err)
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}

		//レスポンスデータ
		games = append(games, map[string]interface{}{
			"id":   id,
			"type": typeCode,
			"name": name,
			"rule": rule,
		})
	}

	resp := map[string]interface{}{
		"result": "OK",
		"games":  games,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
