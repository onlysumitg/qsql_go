var socket = null;


// tell Ws that client is leaving
window.onbeforeunload = function () {
                console.log("Leaving"); 
                let jsonData = {};////WsClientPayload
                jsonData["action"] = "left"; 
                socket.send(JSON.stringify(jsonData)) // send left action to web socket
            }



$(document).ready(function () {
        
        //socket = new WebSocket("ws://127.0.0.1:4000/ws/notification");
        socket = new ReconnectingWebSocket(websocketurl, null, {debug: true, reconectInterval: 3000});

        socket.onopen = () =>{
            console.log("Websocket opened.....")
        }

    // to send data to websocket
    //socket.send(JSON.stringify(jsonData));

        socket.onclose = () => {
            console.log("connection closed");
        }


        socket.onerror = error => {
                    console.log("there was an error");
               
                }
        socket.onmessage = msg => {
            let data = JSON.parse(msg.data); // WsNotification
      

            switch (data.action) {
                case "notification":
                       // notify(data.message,data.messagetype)

                        Swal.fire({
                            position: 'top-end',
                            icon: data.messagetype,
                            text: data.message,
                            showConfirmButton: false,
                            timer: 5000
                            })
                            break;
                case "ping":
                        let jsonData = {};////WsClientPayload
                        jsonData["action"] = "pong"; 
                        socket.send(JSON.stringify(jsonData)); // send left action to web socket
                        break;
             }

        }





})