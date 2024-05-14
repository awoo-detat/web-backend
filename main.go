package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/awoo-detat/werewolf/game"
	"github.com/awoo-detat/werewolf/player"

	"github.com/gorilla/websocket"
)

func main() {
	var g *game.Game
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Disable CORS for testing
		},
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request received", "User-Agent", r.Header["User-Agent"], "RemoteAddr", r.RemoteAddr)
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("error upgrading connection", "error", err)
			return
		}
		p := player.NewPlayer(c)
		if g == nil {
			slog.Info("creating game")
			g = game.NewGame(p)
		}
		go p.Play()
	})
	slog.Info("starting server")
	log.Fatal(http.ListenAndServe(":43200", nil))
}
