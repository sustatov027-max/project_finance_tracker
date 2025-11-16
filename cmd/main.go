package main

import (
	"context"
	"finance_tracker/internal/db"
	"finance_tracker/internal/handlers"
	"finance_tracker/internal/models"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		fmt.Printf("%s %s - %v\n", r.Method, r.URL.Path, duration)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerAuth := r.Header.Get("Authorization")
		if headerAuth == "" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(headerAuth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(parts[1], &models.Claims{}, func(t *jwt.Token) (interface{}, error){
			return []byte("secret_key"), nil
		})
		if err != nil{
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.Valid{
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims := token.Claims.(*models.Claims)
		userID := claims.UserID

		ctx := context.WithValue(r.Context(), models.UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	err := db.Init()
	if err != nil {
		fmt.Println("Database connection failed: ", err)
	}

	defer db.DB.Close()

	publicMux := http.NewServeMux()
	privateMux := http.NewServeMux()

	handlers.NewAuthHandler(publicMux)

	handlers.NewTransactionHandler(privateMux)
	handlers.NewAnalyticsHandler(privateMux)

	rootMux := http.NewServeMux()

	rootMux.Handle("/api/transactions/", AuthMiddleware(privateMux))
    rootMux.Handle("/api/analytics/", AuthMiddleware(privateMux))
    rootMux.Handle("/api/auth/", publicMux)

	loggedMux := loggingMiddleware(rootMux)
	server := http.Server{
		Addr:    ":8080",
		Handler: loggedMux,
	}

	fmt.Println("Starting server at port 8080")

	errListen := server.ListenAndServe()
	if errListen != nil {
		fmt.Println("Error starting server: ", errListen)
	}

}
