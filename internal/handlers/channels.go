package handlers

import (
	"context"
	"fmt"
	"neosec/internal/db"
	"neosec/internal/models"
	"neosec/internal/ws"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var HubInstance *ws.Hub

func GetChannel(w http.ResponseWriter, r *http.Request) {
	channelIDStr := chi.URLParam(r, "channelID")
	channelID, err := primitive.ObjectIDFromHex(channelIDStr)
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	var channel models.Channel
	err = db.ChannelsCollection.FindOne(context.Background(), bson.M{"_id": channelID}).Decode(&channel)
	if err != nil {
		http.Error(w, "Channel not found", http.StatusNotFound)
		return
	}

	// Fetch messages
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(50)
	cursor, err := db.MessagesCollection.Find(context.Background(), bson.M{"channel_id": channelID}, opts)
	var messages []models.Message
	if err == nil {
		cursor.All(context.Background(), &messages)
	}

	// Reverse messages for display (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	w.Header().Set("Content-Type", "text/html")

	// Print Header
	fmt.Fprintf(w, `<div class="chat-header"># %s</div>`, channel.Name)

	// Print Messages
	fmt.Fprintf(w, `<div class="chat-messages" id="chat-messages" hx-swap-oob="true">`)
	for _, msg := range messages {
		var author models.User
		db.UsersCollection.FindOne(context.Background(), bson.M{"_id": msg.AuthorID}).Decode(&author)

		fmt.Fprintf(w, `
			<div class="message">
				<img src="%s" alt="avatar" class="message-avatar">
				<div class="message-content">
					<div class="message-author">%s <span class="message-time">%s</span></div>
					<div class="message-text">%s</div>
				</div>
			</div>
		`, author.Avatar, author.Username, msg.CreatedAt.Format("01/02/2006 15:04"), msg.Content)
	}
	fmt.Fprintf(w, `</div>`)

	// Print Input Area and script to subscribe
	fmt.Fprintf(w, `
	<div class="chat-input-area">
		<form hx-post="/api/v9/channels/%s/messages" hx-target="#chat-messages" hx-swap="beforeend" onsubmit="setTimeout(() => { this.reset(); }, 10);">
			<input type="text" name="content" placeholder="Message #%s" required autocomplete="off" autofocus>
		</form>
	</div>
	<script>
		if (window.wsConn && window.wsConn.readyState === WebSocket.OPEN) {
			window.wsConn.send(JSON.stringify({type: "subscribe", channel_id: "%s"}));
		} else {
			window.currentChannelID = "%s";
		}
		var msgDiv = document.getElementById("chat-messages");
		if (msgDiv) {
			msgDiv.scrollTop = msgDiv.scrollHeight;
		}
	</script>
	`, channelID.Hex(), channel.Name, channelID.Hex(), channelID.Hex())
}

func PostMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)
	channelIDStr := chi.URLParam(r, "channelID")
	channelID, err := primitive.ObjectIDFromHex(channelIDStr)
	if err != nil {
		http.Error(w, "Invalid channel ID", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Content cannot be empty", http.StatusBadRequest)
		return
	}

	msg := models.Message{
		ID:        primitive.NewObjectID(),
		ChannelID: channelID,
		AuthorID:  userID,
		Content:   content,
		CreatedAt: time.Now(),
	}

	_, err = db.MessagesCollection.InsertOne(context.Background(), msg)
	if err != nil {
		http.Error(w, "Error saving message", http.StatusInternalServerError)
		return
	}

	var author models.User
	db.UsersCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&author)
	msg.Author = &author

	// Broadcast to WebSocket
	if HubInstance != nil {
		HubInstance.Broadcast <- &msg
	}

	// Also return HTML to append for the sender
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
		<div class="message">
			<img src="%s" alt="avatar" class="message-avatar">
			<div class="message-content">
				<div class="message-author">%s <span class="message-time">%s</span></div>
				<div class="message-text">%s</div>
			</div>
		</div>
		<script>
			var msgDiv = document.getElementById("chat-messages");
			if (msgDiv) {
				msgDiv.scrollTop = msgDiv.scrollHeight;
			}
		</script>
	`, author.Avatar, author.Username, msg.CreatedAt.Format("01/02/2006 15:04"), msg.Content)
}
