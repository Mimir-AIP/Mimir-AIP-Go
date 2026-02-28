package ws

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for simplicity; tighten in production
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSHandler returns an http.HandlerFunc that upgrades HTTP connections to WebSocket
// and registers the client with the hub.
func WSHandler(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws: upgrade error: %v", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
