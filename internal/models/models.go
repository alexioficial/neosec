package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Username  string               `bson:"username" json:"username"`
	Email     string               `bson:"email" json:"email"`
	Password  string               `bson:"password" json:"-"`
	Avatar    string               `bson:"avatar" json:"avatar"`
	Bio       string               `bson:"bio" json:"bio"`
	Friends   []primitive.ObjectID `bson:"friends" json:"friends"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
}

type Guild struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name      string               `bson:"name" json:"name"`
	OwnerID   primitive.ObjectID   `bson:"owner_id" json:"owner_id"`
	Icon      string               `bson:"icon" json:"icon"`
	Members   []primitive.ObjectID `bson:"members" json:"members"`
	CreatedAt time.Time            `bson:"created_at" json:"created_at"`
}

type Channel struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	GuildID   primitive.ObjectID `bson:"guild_id" json:"guild_id"`
	Name      string             `bson:"name" json:"name"`
	Type      string             `bson:"type" json:"type"` // "text", "voice", "direct"
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// Para chats directos, usaremos un modelo de canal con type="direct" y guardaremos los participantes.
type DirectChannel struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Type         string               `bson:"type" json:"type"` // "direct"
	Participants []primitive.ObjectID `bson:"participants" json:"participants"`
}

type Message struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChannelID primitive.ObjectID `bson:"channel_id" json:"channel_id"`
	AuthorID  primitive.ObjectID `bson:"author_id" json:"author_id"`
	Author    *User              `bson:"-" json:"author,omitempty"` // For frontend population
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type Invite struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Code      string             `bson:"code" json:"code"`
	GuildID   primitive.ObjectID `bson:"guild_id" json:"guild_id"`
	InviterID primitive.ObjectID `bson:"inviter_id" json:"inviter_id"`
	Uses      int                `bson:"uses" json:"uses"`
	MaxUses   int                `bson:"max_uses" json:"max_uses"` // 0 for unlimited
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// Solicitudes de amistad
type FriendRequest struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FromUserID primitive.ObjectID `bson:"from_user_id" json:"from_user_id"`
	ToUserID   primitive.ObjectID `bson:"to_user_id" json:"to_user_id"`
	Status     string             `bson:"status" json:"status"` // "pending", "accepted", "rejected"
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}
