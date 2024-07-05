package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/awoo-detat/werewolf/game"
	"github.com/awoo-detat/werewolf/player"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type GameDto struct {
	ID            uuid.UUID `json:"id"`
	Roleset       string    `json:"roleset"`
	PlayerCount   int       `json:"playerCount"`
	PlayersNeeded int       `json:"playersNeeded"`
	AlivePlayers  int       `json:"alivePlayers"`
	Leader        string    `json:"leader"`
	GamePhase     int       `json:"gamePhase"`
}

func main() {
	var g *game.Game
	games := make(map[uuid.UUID]*game.Game)
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Disable CORS for testing
		},
	}
	players := make(map[uuid.UUID]*player.Player)
	playersByGame := make(map[uuid.UUID]map[uuid.UUID]*player.Player)

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

	http.HandleFunc("GET /new", func(w http.ResponseWriter, r *http.Request) {
		g = nil
		players = make(map[uuid.UUID]*player.Player)
		http.Redirect(w, r, "https://werewolf.live/", http.StatusOK)
	})

	http.HandleFunc("GET /games", func(w http.ResponseWriter, r *http.Request) {
		gameList := []*GameDto{}
		for _, g := range games {
			dto := &GameDto{
				ID:           g.ID,
				Leader:       g.Leader.Name,
				PlayerCount:  len(g.Players),
				GamePhase:    g.Phase,
				AlivePlayers: len(g.AlivePlayers),
			}
			if g.Roleset != nil {
				dto.Roleset = g.Roleset.Name
				dto.PlayersNeeded = len(g.Roleset.Roles)
			}
			gameList = append(gameList, dto)
		}
		// add in the single game
		if g != nil {
			dto := &GameDto{
				ID:           g.ID,
				Leader:       g.Leader.Name,
				PlayerCount:  len(g.Players),
				GamePhase:    g.Phase,
				AlivePlayers: len(g.AlivePlayers),
			}
			if g.Roleset != nil {
				dto.Roleset = g.Roleset.Name
				dto.PlayersNeeded = len(g.Roleset.Roles)
			}
			gameList = append(gameList, dto)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(gameList); err != nil {
			slog.Error("error encoding response", "error", err)
		}
	})

	http.HandleFunc("POST /games", func(w http.ResponseWriter, r *http.Request) {
		// it's going to be connected "properly" momentarily
		c := player.NewMockCommunicator()
		p := player.NewPlayer(c)
		slog.Info("creating new player", "player", p)
		slog.Info("creating game")
		newGame := game.NewGame(p)
		games[newGame.ID] = newGame
		playersByGame[newGame.ID] = make(map[uuid.UUID]*player.Player)
		playersByGame[newGame.ID][p.ID] = p

		w.Header().Add("Location", fmt.Sprintf("/games/%s?id=%s", newGame.ID, p.ID))
		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("GET /games/{id}", func(w http.ResponseWriter, r *http.Request) {
		rawID := r.PathValue("id")

		id, err := uuid.Parse(rawID)
		if err != nil {
			slog.Warn("error parsing UUID", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		gameByID, ok := games[id]
		if !ok {
			slog.Warn("game not found", "id", rawID)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		playersForGame, ok := playersByGame[id]
		if !ok {
			slog.Warn("game not found", "id", rawID)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		slog.Info("request received", "User-Agent", r.Header["User-Agent"], "RemoteAddr", r.RemoteAddr)
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("error upgrading connection", "error", err)
			fmt.Fprintf(w, "error: %s", err)
			return
		}

		var p *player.Player
		query := r.URL.Query()
		if playerID, err := uuid.Parse(query.Get("id")); err == nil {
			p = playersForGame[playerID]
			if p != nil {
				slog.Info("reconnecting to player", "player", p)
				p.Reconnect(c)
			}
		}
		if p == nil {
			p = player.NewPlayer(c)
			playersByGame[gameByID.ID][p.ID] = p
			slog.Info("creating new player", "player", p)

			gameByID.AddPlayer(p)
		}
		go p.Play()
	})

	slog.Info("starting server")
	log.Fatal(http.ListenAndServe(":43200", nil))
}
