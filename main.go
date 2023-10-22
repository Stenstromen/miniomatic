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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	router := mux.NewRouter()

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
