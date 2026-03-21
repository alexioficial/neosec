package main

import (
	"log"
	"neosec/internal/auth"
	"neosec/internal/config"
	"neosec/internal/db"
	"neosec/internal/handlers"
	"neosec/internal/ws"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectMongoDB(cfg.MongoURI)
	auth.InitSecret(cfg.JWTSecret)

	hub := ws.NewHub()
	handlers.HubInstance = hub
	go hub.Run()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Serve static files
	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	// Auth routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	r.Get("/login", handlers.LoginPage)
	r.Post("/api/v9/auth/login", handlers.Login)
	r.Get("/register", handlers.RegisterPage)
	r.Post("/api/v9/auth/register", handlers.Register)
	r.Get("/logout", handlers.Logout)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)

		r.Get("/app", handlers.AppPage)

		// HTMX API endpoints
		r.Post("/api/v9/guilds", handlers.CreateGuild)
		r.Get("/api/v9/guilds/{guildID}/channels", handlers.GetGuildChannels)
		r.Post("/api/v9/guilds/{guildID}/channels", handlers.CreateChannel)

		// User & Friends
		r.Get("/api/v9/users/@me", handlers.GetMe)
		r.Post("/api/v9/users/@me/relationships", handlers.SendFriendRequest)
		r.Post("/api/v9/users/@me/relationships/{requestID}/accept", handlers.AcceptFriendRequest)
		r.Post("/api/v9/users/@me/relationships/{requestID}/reject", handlers.RejectFriendRequest)
		r.Post("/api/v9/users/@me/channels", handlers.CreateDirectMessage)

		r.Get("/api/v9/channels/{channelID}", handlers.GetChannel)
		r.Post("/api/v9/channels/{channelID}/messages", handlers.PostMessage)

		// WebSocket
		r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
			ws.ServeWs(hub, w, r)
		})
	})

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
