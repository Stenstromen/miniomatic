package model

type Item struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

var Items []Item
