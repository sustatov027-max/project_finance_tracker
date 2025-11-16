package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"finance_tracker/internal/db"
	"finance_tracker/internal/models"
	"finance_tracker/pkg/res"
)

type AuthHandler struct{}

func NewAuthHandler(router *http.ServeMux) {
	handler := &AuthHandler{}
	router.Handle("/api/auth/register", handler.Register())
	router.Handle("/api/auth/login", handler.Login())
}

func (handler *AuthHandler) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var requestBody struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			w.Header().Set("Content-Type", "application/json")

			err := json.NewDecoder(r.Body).Decode(&requestBody)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if requestBody.Email == "" || requestBody.Password == "" {
				http.Error(w, "Email and password are required", http.StatusBadRequest)
				return
			}

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(requestBody.Password), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = db.DB.Exec(`INSERT INTO users (email, password_hash) VALUES ($1, $2);`, requestBody.Email, string(hashedPassword))
			if err != nil {
				if strings.Contains(err.Error(), "users_email_key") {
					http.Error(w, "Email already exists", http.StatusConflict)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var statusResponse struct {
				Status string `json:"status"`
			}
			statusResponse.Status = "registered"
			err = res.Json(w, statusResponse, 201)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func (handler *AuthHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var requestBody struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			err := json.NewDecoder(r.Body).Decode(&requestBody)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			var userID int
			var hashedPassword string
			err = db.DB.QueryRow("SELECT id, password_hash FROM users WHERE email = $1;", requestBody.Email).Scan(&userID, &hashedPassword)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "Invalid email or password", http.StatusUnauthorized)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(requestBody.Password))
			if err != nil {
				http.Error(w, "Invalid email or password", http.StatusUnauthorized)
				return
			}


			claims := &models.Claims{
				UserID: userID,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
					IssuedAt:  jwt.NewNumericDate(time.Now()),
				},
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			jwtSecret := []byte("secret_key")

			tokenString, err := token.SignedString(jwtSecret)
			if err != nil{
				http.Error(w, "Could not generate token", http.StatusInternalServerError)
				return
			}

			err = res.Json(w, map[string]string{"token": tokenString}, 200)
			if err != nil{
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}
