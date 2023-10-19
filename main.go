package main

import (
	"log"
	"net/http"

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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	router := mux.NewRouter()

	router.HandleFunc("/api", controller.GetItems).Methods("GET")
	router.HandleFunc("/api/{id}", controller.GetItem).Methods("GET")
	router.HandleFunc("/api", controller.CreateItem).Methods("POST")
	router.HandleFunc("/items/{id}", controller.UpdateItem).Methods("PUT")
	router.HandleFunc("/items/{id}", controller.UpdateItem).Methods("PATCH")
	router.HandleFunc("/api/{id}", controller.DeleteItem).Methods("DELETE")

	err = http.ListenAndServe(":8080", router)

	if err != nil {
		log.Fatal("Failed to start server: ", err)
	}
	log.Println("Server started on: http://localhost:8080")
}
