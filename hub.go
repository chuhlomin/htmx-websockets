package main

import (
	"bytes"
	"fmt"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Inbound messages from the clients.
	Broadcast chan []byte

	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client

	// Registered clients.
	clients map[*Client]bool

	// Mutex for clients
	mutex sync.Mutex
}

// NewHub creates new Hub
func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run handles communication operations with Hub
func (h *Hub) Run() {
	var clientListChanged bool
	for {
		select {
		case client := <-h.Register:
			log.Printf("Registering client %s", client.name)

			h.mutex.Lock()
			h.clients[client] = true
			clientListChanged = true
			h.mutex.Unlock()

			// h.Broadcast <- []byte("<div id=\"users\">" + h.getCurrentUsers() + "</div>")

		case client := <-h.Unregister:

			log.Printf("Unregistering client %s", client.name)

			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			clientListChanged = true
			h.mutex.Unlock()

			// h.Broadcast <- []byte("<div id=\"users\">" + h.getCurrentUsers() + "</div>")

		case message := <-h.Broadcast:
			h.mutex.Lock()
			for client := range h.clients {
				client.send <- message
			}
			h.mutex.Unlock()
		}

		if clientListChanged {
			go func() {
				h.Broadcast <- []byte("<div id=\"users\">" + h.getCurrentUsers() + "</div>")
				clientListChanged = false
			}()
		}
	}
}

func (h *Hub) getCurrentUsers() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("<div id=\"users\">%d users connected:</div>", len(h.clients)))
	buf.WriteString("<ul>")
	for client := range h.clients {
		buf.WriteString("<li>" + client.name + "</li>")
	}
	buf.WriteString("</ul>")
	return buf.String()
}
