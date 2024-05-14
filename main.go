package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/awoo-detat/werewolf/game"
	"github.com/awoo-detat/werewolf/player"

	"github.com/google/uuid"
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
	players := make(map[uuid.UUID]*player.Player)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request received", "User-Agent", r.Header["User-Agent"], "RemoteAddr", r.RemoteAddr)
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("error upgrading connection", "error", err)
			fmt.Fprintf(w, "error: %s", err)
			return
		}

		var p *player.Player
		query := r.URL.Query()
		if id, err := uuid.Parse(query.Get("id")); err == nil {
			p = players[id]
			if p != nil {
				slog.Info("reconnecting to player", "player", p)
				p.Reconnect(c)
			}
		}
		if p == nil {
			p = player.NewPlayer(c)
			players[p.ID] = p
			slog.Info("creating new player", "player", p)

			if g == nil {
				slog.Info("creating game")
				g = game.NewGame(p)
			} else {
				g.AddPlayer(p)
			}
		}
		go p.Play()
	})

	slog.Info("starting server")
	log.Fatal(http.ListenAndServe(":43200", nil))
}
