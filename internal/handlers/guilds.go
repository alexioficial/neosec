package handlers

import (
	"context"
	"fmt"
	"neosec/internal/db"
	"neosec/internal/models"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateGuild(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)
	name := r.FormValue("name")

	if name == "" {
		http.Error(w, "Name cannot be empty", http.StatusBadRequest)
		return
	}

	guildID := primitive.NewObjectID()
	guild := models.Guild{
		ID:        guildID,
		Name:      name,
		OwnerID:   userID,
		Icon:      "https://ui-avatars.com/api/?name=" + name,
		Members:   []primitive.ObjectID{userID},
		CreatedAt: time.Now(),
	}

	_, err := db.GuildsCollection.InsertOne(context.Background(), guild)
	if err != nil {
		http.Error(w, "Error creating guild", http.StatusInternalServerError)
		return
	}

	// Create a default "general" channel
	channel := models.Channel{
		ID:        primitive.NewObjectID(),
		GuildID:   guildID,
		Name:      "general",
		Type:      "text",
		CreatedAt: time.Now(),
	}
	_, _ = db.ChannelsCollection.InsertOne(context.Background(), channel)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="guild-icon" hx-get="/api/v9/guilds/%s/channels" hx-target="#channels-container" hx-swap="innerHTML">
        <img src="%s" alt="%s">
    </div>`, guild.ID.Hex(), guild.Icon, guild.Name)
}

func GetGuildChannels(w http.ResponseWriter, r *http.Request) {
	guildIDStr := chi.URLParam(r, "guildID")
	guildID, err := primitive.ObjectIDFromHex(guildIDStr)
	if err != nil {
		http.Error(w, "Invalid guild ID", http.StatusBadRequest)
		return
	}

	var guild models.Guild
	err = db.GuildsCollection.FindOne(context.Background(), bson.M{"_id": guildID}).Decode(&guild)
	if err != nil {
		http.Error(w, "Guild not found", http.StatusNotFound)
		return
	}

	cursor, err := db.ChannelsCollection.Find(context.Background(), bson.M{"guild_id": guildID})
	var channels []models.Channel
	if err == nil {
		cursor.All(context.Background(), &channels)
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="guild-header">%s</div>`, guild.Name)
	fmt.Fprintf(w, `<div class="channels-list">`)
	for _, ch := range channels {
		fmt.Fprintf(w, `<div class="channel-item" hx-get="/api/v9/channels/%s" hx-target="#main-chat" hx-swap="innerHTML"># %s</div>`, ch.ID.Hex(), ch.Name)
	}
	fmt.Fprintf(w, `</div>`)
	// Create channel form
	fmt.Fprintf(w, `
	<div class="create-channel-form">
		<form hx-post="/api/v9/guilds/%s/channels" hx-target=".channels-list" hx-swap="beforeend" onsubmit="setTimeout(() => { this.reset(); }, 10);">
			<input type="text" name="name" placeholder="new-channel" required>
			<button type="submit">+</button>
		</form>
	</div>
	`, guildID.Hex())
}

func CreateChannel(w http.ResponseWriter, r *http.Request) {
	guildIDStr := chi.URLParam(r, "guildID")
	guildID, err := primitive.ObjectIDFromHex(guildIDStr)
	if err != nil {
		http.Error(w, "Invalid guild ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name cannot be empty", http.StatusBadRequest)
		return
	}

	channel := models.Channel{
		ID:        primitive.NewObjectID(),
		GuildID:   guildID,
		Name:      name,
		Type:      "text",
		CreatedAt: time.Now(),
	}

	_, err = db.ChannelsCollection.InsertOne(context.Background(), channel)
	if err != nil {
		http.Error(w, "Error creating channel", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="channel-item" hx-get="/api/v9/channels/%s" hx-target="#main-chat" hx-swap="innerHTML"># %s</div>`, channel.ID.Hex(), channel.Name)
}
