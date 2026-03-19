package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"username"`
	Password string             `bson:"password"`
}

type Server struct {
	ID    string `bson:"_id"` // UUID
	Name  string `bson:"name"`
	Owner string `bson:"owner"`
}

type Channel struct {
	ID       string `bson:"_id"` // UUID
	ServerID string `bson:"server_id"`
	Name     string `bson:"name"`
}

type Message struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	ServerID  string             `bson:"server_id"`
	ChannelID string             `bson:"channel_id"`
	Username  string             `bson:"username"`
	Content   string             `bson:"content"`
	CreatedAt primitive.DateTime `bson:"created_at"`
}

type DirectMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	User1     string             `bson:"user1"`
	User2     string             `bson:"user2"`
	Sender    string             `bson:"sender"`
	Content   string             `bson:"content"`
	CreatedAt primitive.DateTime `bson:"created_at"`
}

type WsMessage struct {
	ChatMessage string `json:"chat_message"`
	ChannelID   string `json:"channel_id"`
	ServerID    string `json:"server_id"`
	HEADERS     struct {
		HXRequest string `json:"HX-Request"`
	} `json:"HEADERS"`
}
