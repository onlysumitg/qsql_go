package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/onlysumitg/qsql2/internal/iwebsocket"
)

// ------------------------------------------------------
//
// ------------------------------------------------------
func (app *application) WsHandlers(router *chi.Mux) {
	router.Route("/ws", func(r chi.Router) {
		r.Get("/notification", app.WsNotification)

	})

}

var wsChan = make(chan iwebsocket.WsClientPayload)

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
func (app *application) WsNotification(w http.ResponseWriter, r *http.Request) {
	// upgrade connection to websocket
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WsEndpoint", err)
	}

	log.Println("Client conneted to WsEndpoint")

	// response := &iwebsocket.WsServerPayload{Message: "Heloo", MessageType: "start"}

	// err = ws.WriteJSON(response)
	// if err != nil {
	// 	log.Println("WsEndpoint 2", err)
	// }

	conn := iwebsocket.WebSocketConnection{Conn: ws}

	// after 1st call this GoRoutine will process all websocket requests.
	go ListenForWs(&conn)

	// ping clien
	go ping(&conn)
}

// ------------------------------------------------------
//
//	get data from web socket and sent to WS channel
//
// ------------------------------------------------------@
func ping(conn *iwebsocket.WebSocketConnection) {

	// ping client --> in reponse client will send pong --> check ListenToWsChannel()
	var response iwebsocket.WsServerPayload
	response.Action = "ping"
	iwebsocket.BroadcastToOne(*conn, response)
}

// ------------------------------------------------------
//  get data from web socket and sent to WS channel
// ------------------------------------------------------@

// ListenForWs is a goroutine that handles communication between server and client, and
// feeds data into the wsChan
func ListenForWs(conn *iwebsocket.WebSocketConnection) {

	// to recover from panics
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload iwebsocket.WsClientPayload

	for {
		err := conn.ReadJSON(&payload)
		if err != nil { // means connection closed
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
		} else {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

// ------------------------------------------------------
//
//	get data from   WS channel and process
//
// ------------------------------------------------------@
// ListenToWsChannel is a goroutine that waits for an entry on the wsChan, and handles it according to the
// specified action
func ListenToWsChannel() {
	var response iwebsocket.WsServerPayload

	for {
		e := <-wsChan

		switch e.Action {
		case "pong":
			// // get a list of all users and send it back via broadcast
			log.Println("Ws is ready")
			iwebsocket.Clients[e.Conn] = e.Username
			// users := getUserList()
			// response.Action = "notification"
			// response.Message = "Websocket connection is sucessful."
			// response.MessageType = "success"

			// iwebsocket.BroadcastToOne(e.Conn, response)

		case "left":
			// // handle the situation where a user leaves the page
			// response.Action = "list_users"
			delete(iwebsocket.Clients, e.Conn)
			// users := getUserList()
			// response.ConnectedUsers = users
			//iwebsocket.BroadcastToAll(response)

		case "broadcast":
			response.Action = "broadcast"
			response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.Username, e.Message)
			//iwebsocket.BroadcastToAll(response)
		}
	}
}
