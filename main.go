package main

import (
	"log"
	"net/http"
	"os"

	"neosec/db"
	"neosec/handlers"
	"neosec/ws"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	db.InitDB(mongoURI)

	hub := ws.NewHub()
	go hub.Run()

	handlers.InitHandlers(hub)

	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/login", handlers.LoginHandler)
	http.HandleFunc("/register", handlers.RegisterHandler)
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/api/servers", handlers.CreateServerHandler)
	http.HandleFunc("/api/channels", handlers.CreateChannelHandler)
	http.HandleFunc("/api/friends", handlers.CreateFriendRequestHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(hub, w, r)
	})

	log.Println("Server starting on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
