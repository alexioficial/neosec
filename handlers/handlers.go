package handlers

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"neosec/db"
	"neosec/models"
	"neosec/ws"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

var tmpl *template.Template
var HubInstance *ws.Hub

var (
	loginLimiters    = make(map[string]*rate.Limiter)
	registerLimiters = make(map[string]*rate.Limiter)
	mu               sync.Mutex
)

func getLimiter(ip string, limiters map[string]*rate.Limiter, r rate.Limit, b int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	limiter, exists := limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(r, b)
		limiters[ip] = limiter
	}
	return limiter
}

func InitHandlers(hub *ws.Hub) {
	HubInstance = hub
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	cookie, err := r.Cookie("username")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	me := cookie.Value

	serverID := r.URL.Query().Get("server")
	channelID := r.URL.Query().Get("channel")

	if serverID == "" {
		serverID = "@me"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{"created_at", 1}}).SetLimit(50)
	var messages []models.Message
	var friends []string

	if serverID == "@me" {
		// Fetch friends
		cursorF, err := db.FriendsCollection.Find(ctx, bson.M{
			"$or": []bson.M{{"user1": me}, {"user2": me}},
		})
		if err == nil {
			var fDocs []models.Friend
			cursorF.All(ctx, &fDocs)
			for _, f := range fDocs {
				if f.User1 == me {
					friends = append(friends, f.User2)
				} else {
					friends = append(friends, f.User1)
				}
			}
		}

		if channelID != "" {
			u1, u2 := me, channelID
			if u1 > u2 {
				u1, u2 = u2, u1
			}
			cursor, err := db.DirectMessagesCollection.Find(ctx, bson.M{"user1": u1, "user2": u2}, opts)
			if err == nil {
				var dms []models.DirectMessage
				cursor.All(ctx, &dms)
				for _, dm := range dms {
					messages = append(messages, models.Message{
						Username:  dm.Sender,
						Content:   dm.Content,
						CreatedAt: dm.CreatedAt,
					})
				}
			}
		}
	} else {
		// Fetch server messages
		if channelID == "" {
			channelID = "general"
		}
		cursor, err := db.MessagesCollection.Find(ctx, bson.M{"server_id": serverID, "channel_id": channelID}, opts)
		if err == nil {
			cursor.All(ctx, &messages)
		}
	}

	cursorS, _ := db.ServersCollection.Find(ctx, bson.M{})
	var servers []models.Server
	cursorS.All(ctx, &servers)

	var channels []models.Channel
	if serverID != "@me" {
		cursorC, _ := db.ChannelsCollection.Find(ctx, bson.M{"server_id": serverID})
		cursorC.All(ctx, &channels)
	}

	data := struct {
		Username       string
		CurrentServer  string
		CurrentChannel string
		Messages       []models.Message
		Servers        []models.Server
		Channels       []models.Channel
		Friends        []string
	}{
		Username:       me,
		CurrentServer:  serverID,
		CurrentChannel: channelID,
		Messages:       messages,
		Servers:        servers,
		Channels:       channels,
		Friends:        friends,
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
	if r.Method != http.MethodPost {
		return
	}
	friendUsername := r.FormValue("username")
	cookie, _ := r.Cookie("username")
	me := cookie.Value

	if friendUsername == me {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<script>alert('No puedes agregarte a ti mismo')</script>"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := db.UsersCollection.FindOne(ctx, bson.M{"username": friendUsername}).Decode(&user)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<script>alert('Usuario no encontrado')</script>"))
		return
	}

	u1, u2 := me, friendUsername
	if u1 > u2 {
		u1, u2 = u2, u1
	}

	count, _ := db.FriendsCollection.CountDocuments(ctx, bson.M{"user1": u1, "user2": u2})
	if count == 0 {
		db.FriendsCollection.InsertOne(ctx, models.Friend{User1: u1, User2: u2})
	}

	w.Header().Set("HX-Redirect", "/?server=@me&channel="+friendUsername)
}
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		err := tmpl.ExecuteTemplate(w, "login.html", nil)
		if err != nil {
			log.Println("Error executing login template:", err)
		}
		return
	}

	ip := strings.Split(r.RemoteAddr, ":")[0]
	// 10 per minute
	limiter := getLimiter(ip, loginLimiters, rate.Every(time.Minute/10), 10)
	if !limiter.Allow() {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Demasiados intentos. Espera un minuto."))
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := db.UsersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("Usuario o contraseña incorrectos"))
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Error de BD"))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Usuario o contraseña incorrectos"))
		return
	}

	setLoginCookie(w, username)
	w.Header().Set("HX-Redirect", "/")
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		err := tmpl.ExecuteTemplate(w, "register.html", nil)
		if err != nil {
			log.Println("Error executing register template:", err)
		}
		return
	}

	ip := strings.Split(r.RemoteAddr, ":")[0]
	// 5 per minute
	limiter := getLimiter(ip, registerLimiters, rate.Every(time.Minute/5), 5)
	if !limiter.Allow() {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Demasiados intentos de registro. Espera un minuto."))
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password != confirmPassword {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Las contraseñas no coinciden"))
		return
	}

	if len(password) < 6 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("La contraseña debe tener al menos 6 caracteres"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("Error interno"))
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
