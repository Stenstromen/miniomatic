package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/stenstromen/miniomatic/db"
	"github.com/stenstromen/miniomatic/k8sclient"
	"github.com/stenstromen/miniomatic/madmin"
	"github.com/stenstromen/miniomatic/model"
	"github.com/stenstromen/miniomatic/rnd"
	"k8s.io/apimachinery/pkg/api/resource"
)

func GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := db.GetAllData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(items)

}

func GetItem(w http.ResponseWriter, r *http.Request) {
	item, err := db.GetDataByID(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(item)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	var post model.Post
	if r.ContentLength == 0 {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	_ = json.NewDecoder(r.Body).Decode(&post)
	// Validate Storage format
	validStorageFormat := regexp.MustCompile(`^[0-9]+(Ki|Mi|Gi)$`)
	if !validStorageFormat.MatchString(post.Storage) {
		http.Error(w, "Invalid storage format. Expected format: [Number][Ki|Mi|Gi]", http.StatusBadRequest)
		return
	}

	// Try parsing the value using Kubernetes resource package to ensure it's a valid quantity
	_, err := resource.ParseQuantity(post.Storage)
	if err != nil {
		http.Error(w, "Invalid storage value", http.StatusBadRequest)
		return
	}
	randnum := rnd.RandomString(false, 6)
	RootUser := rnd.RandomString(true, 16)
	RootPassword := rnd.RandomString(true, 16)
	AccessKey := rnd.RandomString(true, 17)
	SecretKey := rnd.RandomString(true, 33)
	ClusterIssuer := os.Getenv("CLUSTERISSUER")
	StorageClassName := os.Getenv("STORAGECLASSNAME")
	Storage := post.Storage

	go func() {
		err := k8sclient.CreateMinioResources(randnum, RootUser, RootPassword, ClusterIssuer, StorageClassName, Storage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = madmin.Madmin(randnum, RootUser, RootPassword, post.Bucket, AccessKey, SecretKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}()

	var resp model.Resp
	resp.Status = "provisioning"
	resp.ID = randnum
	resp.Storage = post.Storage
	resp.Bucket = post.Bucket
	resp.URL = "https://" + randnum + "." + os.Getenv("WILDCARD_DOMAIN")
	resp.AccessKey = AccessKey
	resp.SecretKey = SecretKey
	db.InsertData(randnum, post.Bucket, Storage)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range model.Items {
		if item.ID == params["id"] {
			model.Items = append(model.Items[:index], model.Items[index+1:]...)
			var updatedItem model.Item
			_ = json.NewDecoder(r.Body).Decode(&updatedItem)
			updatedItem.ID = params["id"]
			model.Items = append(model.Items, updatedItem)
			json.NewEncoder(w).Encode(updatedItem)
			return
		}
	}
	json.NewEncoder(w).Encode(model.Items)
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	err := db.DeleteData(id)
	if err != nil {
		if strings.Contains(err.Error(), "no record found with ID") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Launch a goroutine to delete the Kubernetes resources associated with this ID
	go func() {
		if err := k8sclient.DeleteMinioResources(id); err != nil {
			log.Printf("Error deleting resources for ID %s: %v", id, err)
		}
	}()

	// Immediately respond with 202 Accepted
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "Deletion in progress"})
}
