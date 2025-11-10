// ソロゲーム判定API
// ソロモードでプレイ可能なゲームタイプ一覧を返す。
// SELECT で types テーブルと game_types テーブルを JOIN し、
// mode='ソロ' かつ is_can_play=TRUE のレコードを取得する。

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func SoloGameListHandler(w http.ResponseWriter, r *http.Request) {
	// ---- SQLクエリ ----
	// ソロモード (modes.mode = 'ソロ') かつ is_can_play=TRUE の type を取得
	query := `
		SELECT t.id, g.code, t.name, t.rule
		FROM types t
		JOIN game_types g ON t.game_type_id = g.id
		WHERE t.mode_id = (SELECT id FROM modes WHERE mode = 'ソロ')
		  AND t.is_can_play = TRUE
	`

	// ---- DB問い合わせ ----
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[ERROR] solo select failed: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// ---- 結果のスキャン ----
	var games []map[string]interface{}
	for rows.Next() {
		var id int
		var typeCode, displayName, rule string

		// 1行分のデータを変数に取り込み
		if err := rows.Scan(&id, &typeCode, &displayName, &rule); err != nil {
			log.Printf("[ERROR] solo scan failed: %v", err)
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}

		// map にして append
		games = append(games, map[string]interface{}{
			"id":   id,
			"type": typeCode,
			"name": displayName,
			"rule": rule,
		})
	}

	// ---- イテレーション中のエラー確認 ----
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] rows iteration failed: %v", err)
		http.Error(w, "Data fetch error", http.StatusInternalServerError)
		return
	}

	// ---- レスポンス作成 ----
	resp := map[string]interface{}{
		"result": "OK",
		"games":  games,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
