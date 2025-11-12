//アカウント作成API

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// 欲しい情報
// name:     表示名（ユニーク）
// password: 明示設定する場合のパスワード（AutoFlg=false時に必須）
// auto_flg: 自動生成/ゲスト想定フラグ（true時はパスワード未設定で作成）
type CreateAccountRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	AutoFlg  bool   `json:"auto_flg"`
}

// JWT署名用シークレットキー
var jwtSecret = []byte("your_secret_key")

// CreateAccountHandler はユーザーアカウントを新規作成し、24時間有効のJWTを返す。
func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	//Headersがjsonの場合
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request (json)", http.StatusBadRequest)
			return
		}
		//Headersがx-www-form-urlencodedの場合
	} else if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid request (form)", http.StatusBadRequest)
			return
		}
		req.Name = r.FormValue("name")
		req.Password = r.FormValue("password")
		req.AutoFlg = r.FormValue("auto_flg") == "true"
		//それ以外
	} else {
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}
	// バリデーションのメッセージ追加
	// 名前がない場合、パスワードがない場合、名前が使われている場合
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "名前の入力をしてください", http.StatusBadRequest)
		return
	}
	if !req.AutoFlg && strings.TrimSpace(req.Password) == "" {
		http.Error(w, "パスワードの入力をしてください", http.StatusBadRequest)
		return
	}

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE name = ?)", req.Name).Scan(&exists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if exists {
		resp := map[string]interface{}{
			"result":  "NG",
			"message": "この名前は既に使用されています",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
	// ---- パスワードハッシュ化 ----
	// AutoFlg=true の場合は未設定（空文字）で保存（ゲスト/自動アカウントを想定）
	var hashedPassword string
	if req.AutoFlg {
		hashedPassword = ""
	} else {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Password hashing failed", http.StatusInternalServerError)
			return
		}
		hashedPassword = string(hash)
	}

	now := time.Now()
	// ---- users にレコード挿入 ----
	result, err := db.Exec(`
		INSERT INTO users (name, password, created_at, last_login)
		VALUES (?, ?, ?, ?)`,
		req.Name, hashedPassword, now, now)
	if err != nil {
		http.Error(w, "ユーザーの作成に失敗しました", http.StatusInternalServerError)
		return
	}
	// 生成されたユーザーIDを取得

	userID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "ユーザーの作成に失敗しました", http.StatusInternalServerError)
		return
	}

	// settings テーブルに初期データを挿入
	_, err = db.Exec(`
		INSERT INTO settings (user_id, bgm_volume, se_volume, icon)
		VALUES (?, ?, ?, ?)`,
		userID, 1.0, 1.0, "default_icon")
	if err != nil {
		http.Error(w, "設定データの作成に失敗しました", http.StatusInternalServerError)
		return
	}
	// tips 初期データ挿入
	_, err = db.Exec(`
		INSERT INTO tips (user_id, solo_tip_count, multi_tip_count, updated_at)
		VALUES (?, ?, ?, ?)`,
		userID, 10000, 10000, now)
	if err != nil {
		http.Error(w, "チップデータの作成に失敗しました", http.StatusInternalServerError)
		return
	}

	// 24時間有効のJWT生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"name":    req.Name,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		http.Error(w, "トークンの作成失敗", http.StatusInternalServerError)
		return
	}

	// 成功時は result=OK と JWT を返す
	resp := map[string]interface{}{
		"result": "OK",
		"token":  signedToken,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
