package main

import (
	"api/internal/handlers"
	"api/internal/middleware"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// getLocalIP はユーザーのホームディレクトリ直下の ip.txt を読み取り、
// そこに書かれているIPアドレス文字列を返す。
// 例: ip.txt の中身が "192.168.0.10" の場合、DB接続先に利用される。
func getLocalIP() (string, error) {
	homeDir := "/home/user1"
	if homeDir == "" {
		return "", fmt.Errorf("環境変数HOMEが設定されていません")
	}

	path := filepath.Join(homeDir, "ip.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ★ 固定したいDBのホストIP
// const dbHost = "192.168.10.1" // ←ここをあなたの固定IPに
//const dbHost = "192.168.56.102" // ←ここをあなたの固定IPに

func main() {
	// ---- DB接続先のIPをファイルから取得 ----
	ip, err := getLocalIP()
	if err != nil {
		log.Fatalf("IPファイル読み込み失敗: %v", err)
	}

	// ---- DSNを生成（ユーザー名/パスワード/ホスト/DB名）----
	// MEMO: 本番運用では環境変数やSecret管理を推奨（ハードコード回避）
	dsn := fmt.Sprintf("db_user1:db_user1@tcp(%s:3306)/graduationdb", ip)
	//dsn := fmt.Sprintf("db_user1:db_user1@tcp(%s:3306)/graduationdb", dbHost)

	log.Println("DSN:", dsn)

	// ---- DBオープンと疎通確認 ----
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Println("DB接続成功")

	// ---- handlers パッケージでグローバルDBを使う場合の初期化 ----
	// CreateAccountHandler 等がパッケージ内グローバル変数 db を参照する設計に対応
	handlers.InitDB(db) // もしあれば

	// ---- ルーター設定（gorilla/mux）----
	r := mux.NewRouter()

	// ========== 公開API（JWT認証不要） ==========
	// アカウント作成/ログイン/メインデータ/ランク・ゲームモード一覧/ソロ・フレンドゲーム一覧/WebSocket（ブラックジャック）
	// MEMO: メソッド制限が必要なら .Methods("POST") 等を付与
	r.HandleFunc("/api/create_account", handlers.CreateAccountHandler)
	r.HandleFunc("/api/login", handlers.LoginHandler)
	r.HandleFunc("/api/get_main_data", handlers.GetMainDataHandler)
	r.HandleFunc("/api/rank_mode", handlers.RankModeHandler)
	r.HandleFunc("/api/game_mode", handlers.GameModeHandler)
	r.HandleFunc("/api/solo_games", handlers.SoloGameListHandler)
	r.HandleFunc("/api/friend_games", handlers.FriendGameListHandler)

	// ========== 認証必須API（JWTミドルウェアで保護） ==========
	// 依存注入（dbを引数で渡す）パターンのハンドラは http.Handler/Func を生成して渡す
	// ルーム作成
	r.Handle("/api/create_room",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.CreateRoomHandler(db))))
	// ルーム参加
	r.Handle("/api/join_room",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.JoinRoomHandler(db))))
	// ルームwebsocket
	r.Handle("/api/ws/room/{room_code}",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.GameRoomWebSocketHandler(db))))
	// ルーム退出
	r.Handle("/api/rooms/leave",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.LeaveRoomHandler(db)))).Methods("POST", "OPTIONS")
	r.Handle("/api/rooms/start",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.StartRoomHandler(db)))).
		Methods("POST", "OPTIONS")
	// ルーム内プレイヤー取得（GET限定）
	r.HandleFunc("/api/rooms/{room_id}/players", handlers.GetPlayersForGameHandler(db)).Methods("GET")
	// ブラックジャックゲーム
	r.Handle(
		"/api/ws/blackjackwebsocket/{room_code}",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.BlackjackWebSocketHandle(db))),
	)
	// 設定更新（ハンドラ側でグローバルdbを使う設計）
	r.Handle("/api/update_settings",
		middleware.JWTMiddleware(http.HandlerFunc(handlers.UpdateUserSettingsHandler)))
	// ソロチップ更新（依存注入）
	r.Handle("/api/updatesolotip",
		middleware.JWTMiddleware(handlers.UpdateSoloTipHandler(db)))

	// 所持チップ取得（GET限定・依存注入）
	r.Handle("/api/get_chip_data",
		middleware.JWTMiddleware(handlers.GetChipDataHandler(db))).Methods("GET")

	// ---- サーバー起動 ----
	//log.Println("サーバー起動: 0.0.0.0:8080")
	//log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))
	log.Println("サーバー起動: 0.0.0.0:9090")
	log.Fatal(http.ListenAndServe("0.0.0.0:9090", r))
}
