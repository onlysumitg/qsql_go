package iwebsocket

import (
	"log"

	"github.com/gorilla/websocket"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
// WebSocketConnection is a wrapper for our websocket connection, in case
// we ever need to put more data into the struct
type WebSocketConnection struct {
	*websocket.Conn
}

// WsPayload defines the websocket request from the client
type WsClientPayload struct {
	Action   string              `json:"action"`
	Username string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

var Clients = make(map[WebSocketConnection]string)

// ------------------------------------------------------
//
// ------------------------------------------------------
type WsServerPayload struct {
	Action      string `json:"action"`
	Message     string `json:"message"`
	MessageType string `json:"messagetype"`
}

// ------------------------------------------------------
//
// ------------------------------------------------------
// broadcastToAll sends ws response to all connected clients
func BroadcastNotification(message, messageType string) {
	serverPayload := WsServerPayload{
		Action:      "notification",
		Message:     message,
		MessageType: messageType,
	}
	BroadcastToAll(serverPayload)
}

// ------------------------------------------------------
//
// ------------------------------------------------------
// broadcastToAll sends ws response to all connected clients
func BroadcastToAll(response WsServerPayload) {
	for client := range Clients {
		err := client.WriteJSON(response)
		if err != nil {
			// the user probably left the page, or their connection dropped
			log.Println("websocket err")
			_ = client.Close()
			delete(Clients, client)
		}
	}
}

// ------------------------------------------------------
//
// ------------------------------------------------------
// broadcastToAll sends ws response to all connected clients
func BroadcastToOne(conn WebSocketConnection, response WsServerPayload) {

	err := conn.WriteJSON(response)
	if err != nil {
		// the user probably left the page, or their connection dropped
		log.Println("websocket err")
		_ = conn.Close()
		delete(Clients, conn)
	}

}
