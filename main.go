package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader configures the parameters for upgrading an HTTP connection to a WebSocket connection.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // Buffer size for reading WebSocket messages
	WriteBufferSize: 1024, // Buffer size for writing WebSocket messages
	CheckOrigin: func(r *http.Request) bool {
		return true // This allows WebSocket connections from any origin (for simplicity)
	},
}

// Client represents a single connection with the WebSocket.
type Client struct {
	conn *websocket.Conn // Each client has a WebSocket connection
}

// 'clients' keeps track of all connected WebSocket clients.
var clients = make(map[*Client]bool)

// 'broadcast' is a channel that is used to pass messages from one client to all other clients.
var broadcast = make(chan []byte)

// handleConnections upgrades HTTP requests to WebSocket connections and manages communication with clients.
func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade the incoming HTTP request to a WebSocket connection.
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err) // If there is an error during upgrading, log it and exit.
	}
	defer ws.Close() // Ensure that the connection is closed when the function exits.

	// Create a new client and store its WebSocket connection.
	client := &Client{conn: ws}
	clients[client] = true // Add the new client to the list of connected clients.

	// Listen for incoming messages from this client.
	for {
		// Read a message from the WebSocket connection.
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("error: %v", err) // Log any errors during message reading.
			delete(clients, client)      // Remove the client from the list of active clients.
			break                        // Exit the loop when the client disconnects.
		}
		// Send the received message to the broadcast channel.
		broadcast <- msg
	}
}

// handleMessages listens for messages on the broadcast channel and sends them to all connected clients.
func handleMessages() {
	for {
		// Wait for a message on the broadcast channel.
		msg := <-broadcast
		// Loop through all connected clients and send the message to each client.
		for client := range clients {
			err := client.conn.WriteMessage(websocket.TextMessage, msg) // Send the message to the client.
			if err != nil {
				log.Printf("error: %v", err) // Log any errors that occur while sending the message.
				client.conn.Close()          // Close the connection if there's an error.
				delete(clients, client)      // Remove the client from the list of active clients.
			}
		}
	}
}

func main() {
	// Define the WebSocket endpoint to handle connections.
	http.HandleFunc("/ws", handleConnections)

	// Start a goroutine to continuously listen for messages and broadcast them.
	go handleMessages()

	// Start the HTTP server on port 8080.
	fmt.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) // If the server fails to start, log the error.
}
