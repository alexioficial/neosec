package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"neosec/db"
	"neosec/models"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	send     chan []byte
	Username string
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	cookie, err := r.Cookie("username")
	username := "Anonymous"
	if err == nil {
		username = cookie.Value
	}

	client := &Client{Hub: hub, Conn: conn, send: make(chan []byte, 256), Username: username}
	client.Hub.Register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var wsMsg models.WsMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Println("Error unmarshaling message:", err)
			continue
		}

		if wsMsg.ChatMessage == "" {
			continue
		}

		channelID := wsMsg.ChannelID
		serverID := wsMsg.ServerID

		if channelID == "" {
			channelID = "general"
		}
		if serverID == "" {
			serverID = "@me"
		}

		if serverID == "@me" {
			friend := channelID
			u1, u2 := c.Username, friend
			if u1 > u2 {
				u1, u2 = u2, u1
			}
			newMsg := models.DirectMessage{
				User1:     u1,
				User2:     u2,
				Sender:    c.Username,
				Content:   wsMsg.ChatMessage,
				CreatedAt: primitive.NewDateTimeFromTime(time.Now()),
			}
			db.DirectMessagesCollection.InsertOne(context.Background(), newMsg)

			tmpl := `<div hx-swap-oob="beforeend:#chat-messages-@me-%s">
            <div class="message">
                <div class="message-header">
                    <span class="message-author">%s</span>
                    <span class="message-timestamp">%s</span>
                </div>
                <div class="message-content">%s</div>
            </div>
        </div>
		<div hx-swap-oob="beforeend:#chat-messages-@me-%s">
            <div class="message">
                <div class="message-header">
                    <span class="message-author">%s</span>
                    <span class="message-timestamp">%s</span>
                </div>
                <div class="message-content">%s</div>
            </div>
        </div>`
			htmlStr := fmt.Sprintf(tmpl, friend, c.Username, time.Now().Format("15:04"), wsMsg.ChatMessage, c.Username, c.Username, time.Now().Format("15:04"), wsMsg.ChatMessage)
			c.Hub.Broadcast <- []byte(htmlStr)
		} else {
			newMsg := models.Message{
				ServerID:  serverID,
				ChannelID: channelID,
				Username:  c.Username,
				Content:   wsMsg.ChatMessage,
				CreatedAt: primitive.NewDateTimeFromTime(time.Now()),
			}
			_, err = db.MessagesCollection.InsertOne(context.Background(), newMsg)
			if err != nil {
				log.Println("Error saving message:", err)
			}

			tmpl := `<div hx-swap-oob="beforeend:#chat-messages-%s-%s">
				<div class="message">
					<div class="message-header">
						<span class="message-author">%s</span>
						<span class="message-timestamp">%s</span>
					</div>
					<div class="message-content">%s</div>
				</div>
			</div>`
			htmlStr := fmt.Sprintf(tmpl, serverID, channelID, c.Username, time.Now().Format("15:04"), wsMsg.ChatMessage)

			c.Hub.Broadcast <- []byte(htmlStr)
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}
