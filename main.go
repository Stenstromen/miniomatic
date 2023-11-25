package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/stenstromen/miniomatic/controller"
	"github.com/stenstromen/miniomatic/db"
)

const APIVersion = "/v1"

func init() {
	log.Println("Initializing Miniomatic...")
	log.Println("Creating database...")
	err := db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
}

func loadEnv() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("error loading .env file or not found: %w. Using default environment variables", err)
	}
	return nil
}

func setupRouter() *mux.Router {
	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.Use(apiKeyMiddleware)

	router.HandleFunc(APIVersion+"/instances", controller.GetItems).Methods("GET")
	router.HandleFunc(APIVersion+"/instances/{id}", controller.GetItem).Methods("GET")
	router.HandleFunc(APIVersion+"/instances", controller.CreateItem).Methods("POST")
	router.HandleFunc(APIVersion+"/instances/{id}", controller.UpdateItem).Methods("PATCH")
	router.HandleFunc(APIVersion+"/instances/{id}", controller.DeleteItem).Methods("DELETE")

	return router
}

func main() {
	if err := loadEnv(); err != nil {
		log.Fatal(err)
	}

	router := setupRouter()

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
		// Additional HTTP server settings can be set here
	}

	go func() {
		log.Println("Server started on: http://:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	gracefulShutdown(server)
}

func gracefulShutdown(server *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Failed to shut down server gracefully: ", err)
	}

	log.Println("Server shut down gracefully")
}

func apiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey, givenApiKey := os.Getenv("API_KEY"), r.Header.Get("X-API-KEY")

		if apiKey == "" {
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
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
