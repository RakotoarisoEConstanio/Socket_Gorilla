package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Message struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Content  string `json:"message"`
	Status   string `json:"status"` // "sent", "received", "seen"
}

var (
	clients   = make(map[*websocket.Conn]string) // Clients connectés
	broadcast = make(chan Message)               // Canal de diffusion des messages
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	mu sync.Mutex // Pour protéger l'accès concurrent
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ws.Close()

	// Ajouter le client à la liste
	mu.Lock()
	clients[ws] = ""
	mu.Unlock()

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			mu.Lock()
			delete(clients, ws)
			mu.Unlock()
			break
		}

		// Mise à jour du statut si c'est une notification de "vu"
		if msg.Status == "seen" {
			// Diffuser l'état "vu" aux clients connectés
			broadcast <- msg
			continue
		}

		// Message envoyé
		msg.Status = "sent"
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		mu.Lock()
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()

	fmt.Println("Server started on :7000")
	err := http.ListenAndServe(":7000", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
