package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
    ID       primitive.ObjectID `bson:"_id,omitempty"`
    Username string             `bson:"username"`
    Password string             `bson:"password"`
}

type Message struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    ChannelID string             `bson:"channel_id"`
    Username  string             `bson:"username"`
    Content   string             `bson:"content"`
    CreatedAt primitive.DateTime `bson:"created_at"`
}

type WsMessage struct {
    ChatMessage string `json:"chat_message"`
    HEADERS     struct {
        HXRequest bool `json:"HX-Request"`
    } `json:"HEADERS"`
}
