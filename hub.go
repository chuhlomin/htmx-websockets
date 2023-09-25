package main

import (
	"bytes"
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

	// Notify channel
	updateList chan struct{}
}

// NewHub creates new Hub
func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		updateList: make(chan struct{}),
	}
}

// Run handles communication operations with Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			log.Printf("Registering client %s", client.name)

			h.mutex.Lock()
			h.clients[client] = true
			go func() {
				h.updateList <- struct{}{}
			}()
			h.mutex.Unlock()

		case client := <-h.Unregister:

			log.Printf("Unregistering client %s", client.name)

			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			go func() {
				h.updateList <- struct{}{}
			}()
			h.mutex.Unlock()

		case message := <-h.Broadcast:
			h.mutex.Lock()
			for client := range h.clients {
				client.send <- message
			}
			h.mutex.Unlock()

		case <-h.updateList:
			h.mutex.Lock()
			for client := range h.clients {
				client.send <- []byte("<div id=\"users\">" + h.getClientList(client.name) + "</div>")
			}
			h.mutex.Unlock()
		}
	}
}

func (h *Hub) getClientList(name string) string {
	var buf bytes.Buffer
	buf.WriteString("<ul>")
	buf.WriteString("<li>" + name + "</li>")
	for client := range h.clients {
		if client.name == name {
			continue
		}
		if !client.active {
			buf.WriteString("<li><em>" + client.name + "</em></li>")
		} else {
			buf.WriteString("<li>" + client.name + "</li>")
		}
	}
	buf.WriteString("</ul>")
	return buf.String()
}
