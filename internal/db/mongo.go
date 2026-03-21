package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client                   *mongo.Client
	UsersCollection          *mongo.Collection
	GuildsCollection         *mongo.Collection
	ChannelsCollection       *mongo.Collection
	MessagesCollection       *mongo.Collection
	FriendRequestsCollection *mongo.Collection
)

func ConnectMongoDB(uri string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	Client = client
	db := client.Database("neosec_discord")

	UsersCollection = db.Collection("users")
	GuildsCollection = db.Collection("guilds")
	ChannelsCollection = db.Collection("channels")
	MessagesCollection = db.Collection("messages")
	FriendRequestsCollection = db.Collection("friend_requests")

	log.Println("Connected to MongoDB successfully!")
}
