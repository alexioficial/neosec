package handlers

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"time"

	"neosec/db"
	"neosec/models"
	"neosec/ws"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var tmpl *template.Template
var HubInstance *ws.Hub

func InitHandlers(hub *ws.Hub) {
	HubInstance = hub
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("username")
	if err != nil {
		err = tmpl.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			log.Println("Error executing login template:", err)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{"created_at", 1}}).SetLimit(50)
	cursor, err := db.MessagesCollection.Find(ctx, bson.M{"channel_id": "general"}, opts)

	var messages []models.Message
	if err == nil {
		cursor.All(ctx, &messages)
	}

	data := struct {
		Username string
		Messages []models.Message
	}{
		Username: cookie.Value,
		Messages: messages,
	}

	err = tmpl.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		log.Println("Error executing index template:", err)
	}
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	action := r.FormValue("action")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if action == "register" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error hasheando password", http.StatusInternalServerError)
			return
		}

		user := models.User{Username: username, Password: string(hashedPassword)}
		_, err = db.UsersCollection.InsertOne(ctx, user)
		if err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("Error creando usuario. Tal vez ya existe."))
			return
		}

		setLoginCookie(w, username)
		w.Header().Set("HX-Redirect", "/")
		return
	}

	var user models.User
	err := db.UsersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("Usuario no encontrado"))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Error de BD"))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Contraseña incorrecta"))
		return
	}

	setLoginCookie(w, username)
	w.Header().Set("HX-Redirect", "/")
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "username",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	w.Header().Set("HX-Redirect", "/")
}

func setLoginCookie(w http.ResponseWriter, username string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "username",
		Value:    username,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})
}
