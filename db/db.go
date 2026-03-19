package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var UsersCollection *mongo.Collection
var MessagesCollection *mongo.Collection
var ServersCollection *mongo.Collection
var ChannelsCollection *mongo.Collection
var FriendsCollection *mongo.Collection
var DirectMessagesCollection *mongo.Collection

func InitDB(uri string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	var err error
	Client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = Client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")

	db := Client.Database("neosec")
	UsersCollection = db.Collection("users")
	MessagesCollection = db.Collection("messages")
	ServersCollection = db.Collection("servers")
	ChannelsCollection = db.Collection("channels")
	FriendsCollection = db.Collection("friends")
	DirectMessagesCollection = db.Collection("direct_messages")
}
