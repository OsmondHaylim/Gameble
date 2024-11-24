package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "github.com/gorilla/websocket"
    "github.com/satori/go.uuid"
)

type ClientManager struct {
    clients         map[*Client]bool    // Active client lists
    broadcast       chan []byte         // Broadcasted messages
    register        chan *Client        // Client to register
    unregister      chan *Client        // Client waiting to be removed
}

type Client struct {
    id              string              // Primary key
    socket          *websocket.Conn     // Socket connection
    send            chan []byte         // Message to send
}
type Message struct {
    sender          string      `json:"sender, omitempty"`
    recipient       string      `json:"recipient, omitempty"`
    content         string      `json:"content, omitempty"`
}

var manager = ClientManager{
    clients:        make(map[*Client]bool),
    broadcast:      make(chan []byte),
    register:       make(chan *Client),
    unregister:     make(chan *Client),
}

func main(){
    fmt.Println("Starting application...")
    go manager.start()
    http.HandleFunc("/ws", wsPage)
    http.ListenAndServe(":12345", nil)
}

func (manager *ClientManager) start() {
    for {
        select {
        case conn := <-manager.register:                                                        
            manager.clients[conn] = true                                                        // Include conn in manager clients list
            jsonMessage, _ := json.Marshal(&Message{content: "/A new socket has connected."})   // Send JSON connect message 
            manager.send(jsonMessage, conn)
        case conn := <-manager.unregister:
            if _, ok := manager.clients[conn]; ok {                                             // If a client isn't in client list
                close(conn.send)                                                                // Close connection channel
                delete(manager.clients, conn)                                                   // Delete client from client list
                jsonMessage, _ := json.Marshal(&Message{content: "/A socket has disconnected."})// Send JSON disconnect message
                manager.send(jsonMessage, conn)
            }
        case message := <-manager.broadcast:
            for conn := range manager.clients {                                                 // Loops every client in client lists
                select {
                case conn.send <- message:                                                      // Send message to send channel in client
                default:
                    close(conn.send)                                                            // Close send channel
                    delete(manager.clients, conn)                                               // Delete client from client list
                }
            }
        }
    }
}

func (manager *ClientManager) send(message []byte, ignore *Client) {                            // Send message to clients.send
    for conn := range manager.clients {
        if conn != ignore {
            conn.send <- message
        }
    }
}

func (c *Client) read() {
    defer func() {                                                                              // Unregister client after read and close socket
        manager.unregister <- c             
        c.socket.Close()
    }()

    for {
        _, message, err := c.socket.ReadMessage()                                               // Read message from WebSocket
        if err != nil {                                                                         // Close socket and unregister client if error
            manager.unregister <- c
            c.socket.Close()
            break
        }
        jsonMessage, _ := json.Marshal(&Message{sender: c.id, content: string(message)})        // Convert message into JSON
        manager.broadcast <- jsonMessage                                                        // Send JSON into broadcast channel
    }
}

func (c *Client) write() {
    defer func() {
        c.socket.Close()
    }()
    for {
        select {
        case message, ok := <-c.send:                                                            // Input send channel to message variable
            if !ok {
                c.socket.WriteMessage(websocket.CloseMessage, []byte{})                          // Send Close message if error
                return
            }
            c.socket.WriteMessage(websocket.TextMessage, message)                                // Send Message
        }
    }
}

func wsPage(res http.ResponseWriter, req *http.Request) {
    conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
    if error != nil {
        http.NotFound(res, req)
        return
    }
    client := &Client{id: uuid.NewV4().String(), socket: conn, send: make(chan []byte)}

    manager.register <- client

    go client.read()
    go client.write()
}
