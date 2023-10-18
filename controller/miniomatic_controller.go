package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/stenstromen/miniomatic/k8sclient"
	"github.com/stenstromen/miniomatic/model"
	"github.com/stenstromen/miniomatic/rnd"
)

func GetItems(w http.ResponseWriter, r *http.Request) {
	pods, err := k8sclient.GetPods()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rand := rnd.RandomString(6)
	fmt.Println(rand)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pods)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for _, item := range model.Items {
		if item.ID == params["id"] {
			json.NewEncoder(w).Encode(item)
			return
		}
	}

	returnitem, err := json.Marshal(&model.Item{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if string(returnitem) == "{}" {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	w.Write(returnitem)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	var item model.Item
	_ = json.NewDecoder(r.Body).Decode(&item)
	model.Items = append(model.Items, item)
	json.NewEncoder(w).Encode(item)
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
	for index, item := range model.Items {
		if item.ID == params["id"] {
			model.Items = append(model.Items[:index], model.Items[index+1:]...)
			break
		}
	}
	json.NewEncoder(w).Encode(model.Items)
}
