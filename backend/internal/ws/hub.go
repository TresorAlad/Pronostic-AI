package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prono/backend/internal/db"
	"github.com/prono/backend/internal/predictions"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) BroadcastEvent(eventType string, data interface{}) {
	msg, _ := json.Marshal(map[string]interface{}{
		"type": eventType,
		"data": data,
		"ts":   time.Now().UTC().Format(time.RFC3339),
	})
	h.broadcast <- msg
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}

type LiveService struct {
	hub         *Hub
	store       *db.Store
	predictions *predictions.Handler
	redis       *redis.Client
}

func NewLiveService(hub *Hub, store *db.Store, pred *predictions.Handler, redis *redis.Client) *LiveService {
	return &LiveService{hub: hub, store: store, predictions: pred, redis: redis}
}

func (ls *LiveService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{hub: ls.hub, conn: conn, send: make(chan []byte, 256)}
	ls.hub.register <- client
	go client.writePump()
	go client.readPump()
}

func (ls *LiveService) StartPolling(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ls.pollLiveMatches(ctx)
		}
	}
}

func (ls *LiveService) pollLiveMatches(ctx context.Context) {
	matches, err := ls.store.GetLiveMatches(ctx)
	if err != nil {
		return
	}

	for _, match := range matches {
		liveKey := "live:match:" + match.ID
		liveData, err := ls.redis.Get(ctx, liveKey).Result()
		if err == nil {
			ls.hub.BroadcastEvent("match_update", json.RawMessage(liveData))
		} else {
			data, _ := json.Marshal(map[string]interface{}{
				"match_id":   match.ID,
				"minute":     match.Minute,
				"home_score": match.HomeScore,
				"away_score": match.AwayScore,
				"home_team":  match.HomeTeam.Name,
				"away_team":  match.AwayTeam.Name,
			})
			ls.hub.BroadcastEvent("match_update", json.RawMessage(data))
		}

		pred, err := ls.predictions.RunLivePrediction(ctx, match.ID)
		if err != nil {
			log.Printf("Live prediction error for %s: %v", match.ID, err)
			continue
		}
		ls.hub.BroadcastEvent("prediction_update", pred)
	}
}
