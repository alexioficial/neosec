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
	fmt.Fprintf(w, `<div class="guild-icon-wrapper" onclick="document.querySelectorAll('.guild-icon-wrapper').forEach(e=>e.classList.remove('active')); this.classList.add('active');">
        <div class="pill"></div>
        <div class="guild-icon" hx-get="/api/v1/guilds/%s/channels" hx-target="#channels-container" hx-swap="innerHTML" title="%s">
            <img src="%s" alt="%s" onerror="this.style.display='none'; this.parentNode.innerHTML='%s'"/>
        </div>
    </div>`, guild.ID.Hex(), guild.Name, guild.Icon, guild.Name, string(guild.Name[0]))
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
	fmt.Fprintf(w, `<div class="guild-header-area">%s <svg width="18" height="18" viewBox="0 0 24 24"><path fill="currentColor" d="M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z"/></svg></div>`, guild.Name)
	fmt.Fprintf(w, `<div class="channels-list">`)
	fmt.Fprintf(w, `<div class="channel-category"><svg width="12" height="12" viewBox="0 0 24 24"><path fill="currentColor" d="M16.59 8.59L12 13.17 7.41 8.59 6 10l6 6 6-6z"/></svg>Text Channels</div>`)
	for _, ch := range channels {
		fmt.Fprintf(w, `<a href="#" class="channel-item" hx-get="/api/v1/channels/%s" hx-target="#main-chat" hx-swap="innerHTML" onclick="document.querySelectorAll('.channel-item').forEach(e=>e.classList.remove('active')); this.classList.add('active');"><svg aria-hidden="true" role="img" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24"><path fill="currentColor" fill-rule="evenodd" d="M10.99 3.16A1 1 0 1 0 9 2.84L8.15 8H4a1 1 0 0 0 0 2h3.82l-.67 4H3a1 1 0 1 0 0 2h3.82l-.8 4.84a1 1 0 0 0 1.97.32L8.85 16h4.97l-.8 4.84a1 1 0 0 0 1.97.32l.86-5.16H20a1 1 0 1 0 0-2h-3.82l.67-4H21a1 1 0 1 0 0-2h-3.82l.8-4.84a1 1 0 1 0-1.97-.32L15.15 8h-4.97l.8-4.84ZM14.82 10h-4.97l-.67 4h4.97l.67-4Z" clip-rule="evenodd"></path></svg>%s</a>`, ch.ID.Hex(), ch.Name)
	}
	fmt.Fprintf(w, `</div>`)
	// Create channel form
	fmt.Fprintf(w, `
	<div style="padding: 0 8px; margin-top: 8px;">
		<form hx-post="/api/v1/guilds/%s/channels" hx-target=".channels-list" hx-swap="beforeend" onsubmit="setTimeout(() => { this.reset(); }, 10);" style="display:flex; gap: 4px;">
			<input type="text" name="name" placeholder="new-channel" required style="flex: 1; padding: 6px; border-radius: 4px; border: none; background: var(--bg-tertiary); color: var(--text-normal);">
			<button type="submit" style="padding: 6px 12px; background: var(--brand); color: white; border: none; border-radius: 4px; cursor:pointer;">+</button>
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
	fmt.Fprintf(w, `<a href="#" class="channel-item" hx-get="/api/v1/channels/%s" hx-target="#main-chat" hx-swap="innerHTML" onclick="document.querySelectorAll('.channel-item').forEach(e=>e.classList.remove('active')); this.classList.add('active');"><svg aria-hidden="true" role="img" xmlns="http://www.w3.org/2000/svg" width="24" height="24" fill="none" viewBox="0 0 24 24"><path fill="currentColor" fill-rule="evenodd" d="M10.99 3.16A1 1 0 1 0 9 2.84L8.15 8H4a1 1 0 0 0 0 2h3.82l-.67 4H3a1 1 0 1 0 0 2h3.82l-.8 4.84a1 1 0 0 0 1.97.32L8.85 16h4.97l-.8 4.84a1 1 0 0 0 1.97.32l.86-5.16H20a1 1 0 1 0 0-2h-3.82l.67-4H21a1 1 0 1 0 0-2h-3.82l.8-4.84a1 1 0 1 0-1.97-.32L15.15 8h-4.97l.8-4.84ZM14.82 10h-4.97l-.67 4h4.97l.67-4Z" clip-rule="evenodd"></path></svg>%s</a>`, channel.ID.Hex(), channel.Name)
}
