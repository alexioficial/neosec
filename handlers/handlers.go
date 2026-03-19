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

	"github.com/google/uuid"
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

	serverID := r.URL.Query().Get("server")
	channelID := r.URL.Query().Get("channel")

	if serverID == "" {
		serverID = "global"
	}
	if channelID == "" {
		channelID = "general"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{"created_at", 1}}).SetLimit(50)
	cursor, err := db.MessagesCollection.Find(ctx, bson.M{"server_id": serverID, "channel_id": channelID}, opts)
	var messages []models.Message
	if err == nil {
		cursor.All(ctx, &messages)
	}

	cursorS, _ := db.ServersCollection.Find(ctx, bson.M{})
	var servers []models.Server
	cursorS.All(ctx, &servers)

	cursorC, _ := db.ChannelsCollection.Find(ctx, bson.M{"server_id": serverID})
	var channels []models.Channel
	cursorC.All(ctx, &channels)

	data := struct {
		Username string
		Messages []models.Message
		Servers  []models.Server
		Channels []models.Channel
	}{
		Username: cookie.Value,
		Messages: messages,
		Servers:  servers,
		Channels: channels,
	}

	err = tmpl.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		log.Println("Error executing index template:", err)
	}
}
func CreateServerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	name := r.FormValue("name")
	cookie, _ := r.Cookie("username")

	serverID := uuid.New().String()
	server := models.Server{ID: serverID, Name: name, Owner: cookie.Value}
	db.ServersCollection.InsertOne(context.Background(), server)

	// Return HTML snippet for HTMX
	shortName := name
	if len(name) > 2 {
		shortName = name[:2]
	}
	w.Header().Set("Content-Type", "text/html")
	html := `<a href="?server=` + serverID + `" class="server-icon">` + shortName + `</a>`
	w.Write([]byte(html))
}

func CreateChannelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	name := r.FormValue("name")
	serverID := r.FormValue("server_id")
	if serverID == "" || serverID == "default" {
		serverID = "global" // Fallback
	}

	channelID := uuid.New().String()
	channel := models.Channel{ID: channelID, Name: name, ServerID: serverID}
	db.ChannelsCollection.InsertOne(context.Background(), channel)

	w.Header().Set("Content-Type", "text/html")
	html := `<a href="?server=` + serverID + `&channel=` + channelID + `" class="channel-item" style="text-decoration: none;"><span class="channel-hash">#</span> ` + name + `</a>`
	w.Write([]byte(html))
}
func CreateFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	// Dummy handler for now
	w.WriteHeader(http.StatusOK)
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
