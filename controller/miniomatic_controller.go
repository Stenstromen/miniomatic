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
	creds := model.Credentials{
		RandNum:      rnd.RandomString(false, 6),
		RootUser:     rnd.RandomString(true, 16),
		RootPassword: rnd.RandomString(true, 16),
	}
	AccessKey, SecretKey, ClusterIssuer, StorageClassName := rnd.RandomString(true, 17), rnd.RandomString(true, 33), os.Getenv("CLUSTERISSUER"), os.Getenv("STORAGECLASSNAME")
	if ClusterIssuer == "" {
		ClusterIssuer = "letsencrypt"
	}
	if StorageClassName == "" {
		StorageClassName = "local-pv"
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

	_, err := resource.ParseQuantity(post.Storage)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid storage value")
		return
	}

	go func() {
		err := k8sclient.CreateMinioResources(creds, ClusterIssuer, StorageClassName, post.Storage)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		err = madmin.Madmin(creds, post.Bucket, AccessKey, SecretKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}()

	resp := model.Resp{
		Status:    "provisioning",
		ID:        creds.RandNum,
		Storage:   post.Storage,
		Bucket:    post.Bucket,
		URL:       "https://" + creds.RandNum + "." + os.Getenv("WILDCARD_DOMAIN"),
		AccessKey: AccessKey,
		SecretKey: SecretKey,
	}

	db.InsertData(creds.RandNum, post.Bucket, post.Storage)
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

	_, err = resource.ParseQuantity(post.Storage)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid storage value")
		return
	}

	k8sclient.ResizeMinioPVC(ID, post.Storage)

	resp := model.Resp{
		Status:  "resizing",
		ID:      ID,
		Storage: post.Storage,
		Bucket:  InitBucket.InitBucket,
		URL:     "https://" + ID + "." + os.Getenv("WILDCARD_DOMAIN"),
	}
	db.UpdateData(ID, resp.Bucket, post.Storage)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)

}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := db.DeleteData(id)
	if err != nil {
		if strings.Contains(err.Error(), "no record found with ID") {
			respondWithError(w, http.StatusNotFound, err.Error())
		} else {
			respondWithError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}

	go func() {
		if err := k8sclient.DeleteMinioResources(id); err != nil {
			log.Printf("Error deleting resources for ID %s: %v", id, err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "Deletion in progress"})
}
