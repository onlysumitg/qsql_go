package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) WsHandlers(router *chi.Mux) {
	router.Route("/ws", func(r chi.Router) {
		r.Get("/notification", app.WsNotification)

	})

}

// ------------------------------------------------------
//
// ------------------------------------------------------
// WebSocketConnection is a wrapper for our websocket connection, in case
// we ever need to put more data into the struct
type WebSocketConnection struct {
	*websocket.Conn
}

// WsPayload defines the websocket request from the client
type WsPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

var clients = make(map[WebSocketConnection]string)

// ------------------------------------------------------
//
// ------------------------------------------------------
var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ------------------------------------------------------
//
// ------------------------------------------------------
type WsNotification struct {
	Message     string `json:"message"`
	MessageType string `json:"messagetype"`
}

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) WsNotification(w http.ResponseWriter, r *http.Request) {
	// upgrade connection to websocket
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WsEndpoint", err)
	}

	log.Println("Client conneted to WsEndpoint")

	response := &WsNotification{Message: "Heloo", MessageType: "start"}

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println("WsEndpoint 2", err)
	}
}

// ------------------------------------------------------
//
// ------------------------------------------------------
// broadcastToAll sends ws response to all connected clients
func broadcastToAll(response WsNotification) {
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			// the user probably left the page, or their connection dropped
			log.Println("websocket err")
			_ = client.Close()
			delete(clients, client)
		}
	}
}
