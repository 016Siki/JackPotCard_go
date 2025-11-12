//メインデータ取得API

package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

func GetMainDataHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		log.Println("[ERROR] handlers.db is nil!")
		http.Error(w, "エラー:サーバーがない", http.StatusInternalServerError)
		return
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// ユーザー名からユーザーIDを取得
	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE name = ?", claims.Username).Scan(&userID)
	if err != nil {
		log.Printf("[ERROR] Failed to get user id for %s: %v", claims.Username, err)
		http.Error(w, "Failed to retrieve user id", http.StatusInternalServerError)
		return
	}

	// user_idでsettingsテーブルから設定を取得
	var bgmVolume, seVolume float64
	var icon string
	err = db.QueryRow("SELECT bgm_volume, se_volume, icon FROM settings WHERE user_id = ?", userID).
		Scan(&bgmVolume, &seVolume, &icon)
	if err != nil {
		if err == sql.ErrNoRows {
			// 設定がなければデフォルト値を返す
			bgmVolume = 0.5
			seVolume = 0.5
			icon = "default_icon"
		} else {
			log.Printf("[ERROR] Failed to get settings for user %s: %v", claims.Username, err)
			http.Error(w, "Failed to retrieve settings", http.StatusInternalServerError)
			return
		}
	}

	// ③ user_idでtipsテーブルからチップ情報を取得
	var soloTip, multiTip int
	err = db.QueryRow("SELECT solo_tip_count, multi_tip_count FROM tips WHERE user_id = ?", userID).
		Scan(&soloTip, &multiTip)
	if err != nil {
		if err == sql.ErrNoRows {
			// チップがなければデフォルト値
			soloTip = 10000
			multiTip = 10000
		} else {
			log.Printf("[ERROR] Failed to get tips for user %s: %v", claims.Username, err)
			http.Error(w, "Failed to retrieve tips", http.StatusInternalServerError)
			return
		}
	}

	response := map[string]interface{}{
		"result":  "OK",
		"message": "Success",
		"settings": map[string]interface{}{
			"bgm_volume": bgmVolume,
			"se_volume":  seVolume,
			"icon":       icon,
			"user_id":    userID,
		},
		"tips": map[string]interface{}{
			"solotip":  soloTip,
			"multitip": multiTip,
		},

		"username": claims.Username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
