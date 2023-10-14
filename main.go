package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stenstromen/miniomatic/controller"
)

func init() {
	log.Println("Initializing Miniomatic...")
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/items", controller.GetItems).Methods("GET")
	router.HandleFunc("/items/{id}", controller.GetItem).Methods("GET")
	router.HandleFunc("/items", controller.CreateItem).Methods("POST")
	router.HandleFunc("/items/{id}", controller.UpdateItem).Methods("PUT")
	router.HandleFunc("/items/{id}", controller.DeleteItem).Methods("DELETE")

	err := http.ListenAndServe(":8080", router)

	if err != nil {
		log.Fatal("Failed to start server: ", err)
	}
	log.Println("Server started on: http://localhost:8080")
}
