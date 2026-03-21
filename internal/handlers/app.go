package handlers

import (
	"context"
	"neosec/internal/db"
	"neosec/internal/models"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func AppPage(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)

	// Get user
	var user models.User
	err := db.UsersCollection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get user's guilds
	cursor, err := db.GuildsCollection.Find(context.Background(), bson.M{"members": userID})
	var guilds []models.Guild
	if err == nil {
		cursor.All(context.Background(), &guilds)
	}

	data := struct {
		User   models.User
		Guilds []models.Guild
	}{
		User:   user,
		Guilds: guilds,
	}

	RenderTemplate(w, "app.html", data)
}
