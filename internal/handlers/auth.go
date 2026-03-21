package handlers

import (
	"context"
	"html/template"
	"neosec/internal/auth"
	"neosec/internal/db"
	"neosec/internal/models"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var templates = template.Must(template.ParseGlob("web/templates/*.html"))

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func LoginPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "login.html", nil)
}

func RegisterPage(w http.ResponseWriter, r *http.Request) {
	RenderTemplate(w, "register.html", nil)
}

func Login(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	var user models.User
	err := db.UsersCollection.FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red;">Invalid email or password</p>`))
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red;">Invalid email or password</p>`))
		return
	}

	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red;">Error generating token</p>`))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(72 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/app")
}

func Register(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Check if user exists
	count, _ := db.UsersCollection.CountDocuments(context.Background(), bson.M{"email": email})
	if count > 0 {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red;">Email already registered</p>`))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red;">Error processing password</p>`))
		return
	}

	newUser := models.User{
		ID:        primitive.NewObjectID(),
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		Avatar:    "https://ui-avatars.com/api/?name=" + username,
		CreatedAt: time.Now(),
	}

	_, err = db.UsersCollection.InsertOne(context.Background(), newUser)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p style="color:red;">Error creating user</p>`))
		return
	}

	token, _ := auth.GenerateToken(newUser.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(72 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	w.Header().Set("HX-Redirect", "/app")
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
