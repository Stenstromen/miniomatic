package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/stenstromen/miniomatic/controller"
	"github.com/stenstromen/miniomatic/db"

	"github.com/gorilla/mux"
)

func init() {
	log.Println("Initializing Miniomatic...")
	log.Println("Creating database...")
	err := db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
}

const APIVersion = "/v1"

func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := os.Getenv("API_KEY")
		if apiKey == "" {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		givenApiKey := r.Header.Get("X-API-KEY")
		if givenApiKey != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, X-API-KEY")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	router := mux.NewRouter()

	router.Use(corsMiddleware)
	router.Use(apiKeyMiddleware)

	router.HandleFunc(APIVersion+"/instances", controller.GetItems).Methods("GET")
	router.HandleFunc(APIVersion+"/instances/{id}", controller.GetItem).Methods("GET")
	router.HandleFunc(APIVersion+"/instances", controller.CreateItem).Methods("POST")
	router.HandleFunc(APIVersion+"/instances/{id}", controller.UpdateItem).Methods("PATCH")
	router.HandleFunc(APIVersion+"/instances/{id}", controller.DeleteItem).Methods("DELETE")

	// Listen for the interrupt signal.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	server := &http.Server{Addr: ":8080", Handler: router}

	log.Println("Server started on: http://:8080")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	<-stop

	// Set a timeout for the graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)
	log.Println("Shutting down gracefully...")
}
