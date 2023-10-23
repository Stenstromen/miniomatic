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

// Common error helper
func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func validateStorageFormat(storage string) bool {
	validStorageFormat := regexp.MustCompile(`^[0-9]+(Ki|Mi|Gi)$`)
	return validStorageFormat.MatchString(storage)
}

func GetItems(w http.ResponseWriter, r *http.Request) {
	items, err := db.GetAllData()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(items) == 0 {
		respondWithError(w, http.StatusNotFound, "No records found")
		return
	}
	json.NewEncoder(w).Encode(items)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	item, err := db.GetDataByID(mux.Vars(r)["id"])
	if item == nil {
		respondWithError(w, http.StatusNotFound, "No record found")
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	json.NewEncoder(w).Encode(item)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	var post model.Post
	if r.ContentLength == 0 {
		respondWithError(w, http.StatusBadRequest, "Empty request body")
		return
	}
	_ = json.NewDecoder(r.Body).Decode(&post)

	if !validateStorageFormat(post.Storage) {
		respondWithError(w, http.StatusBadRequest, "Invalid storage format. Expected format: [Number][Ki|Mi|Gi]")
		return
	}

	// Try parsing the value using Kubernetes resource package to ensure it's a valid quantity
	_, err := resource.ParseQuantity(post.Storage)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid storage value")
		return
	}
	randnum := rnd.RandomString(false, 6)
	RootUser := rnd.RandomString(true, 16)
	RootPassword := rnd.RandomString(true, 16)
	AccessKey := rnd.RandomString(true, 17)
	SecretKey := rnd.RandomString(true, 33)
	ClusterIssuer := os.Getenv("CLUSTERISSUER")
	if ClusterIssuer == "" {
		ClusterIssuer = "letsencrypt"
	}
	StorageClassName := os.Getenv("STORAGECLASSNAME")
	if StorageClassName == "" {
		StorageClassName = "local-pv"
	}
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
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	var post model.Post
	ID := mux.Vars(r)["id"]

	InitBucket, err := db.GetDataByID(ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if r.ContentLength == 0 {
		respondWithError(w, http.StatusBadRequest, "Empty request body")
		return
	}
	_ = json.NewDecoder(r.Body).Decode(&post)
	if !validateStorageFormat(post.Storage) {
		respondWithError(w, http.StatusBadRequest, "Invalid storage format. Expected format: [Number][Ki|Mi|Gi]")
		return
	}

	// Try parsing the value using Kubernetes resource package to ensure it's a valid quantity
	_, err = resource.ParseQuantity(post.Storage)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid storage value")
		return
	}

	k8sclient.ResizeMinioPVC(ID, post.Storage)

	var resp model.Resp
	resp.Status = "resizing"
	resp.ID = ID
	resp.Storage = post.Storage
	resp.Bucket = InitBucket.InitBucket
	resp.URL = "https://" + ID + "." + os.Getenv("WILDCARD_DOMAIN")
	db.UpdateData(ID, resp.Bucket, post.Storage)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)

}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	err := db.DeleteData(id)
	if err != nil {
		if strings.Contains(err.Error(), "no record found with ID") {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
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
