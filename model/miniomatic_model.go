package model

type Item struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

var Items []Item

type Post struct {
	Gi     int    `json:"gi"`
	Bucket string `json:"bucket"`
}

type Resp struct {
	ID        string `json:"id,omitempty"`
	Gi        string `json:"gi,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	URL       string `json:"url,omitempty"`
	AccessKey string `json:"accesskey,omitempty"`
	SecretKey string `json:"secretkey,omitempty"`
}

type Record struct {
	Date       string `json:"date,omitempty"`
	ID         string `json:"id,omitempty"`
	InitBucket string `json:"initbucket,omitempty"`
	URL        string `json:"url,omitempty"`
	StorageGbi int    `json:"storagegbi,omitempty"`
}
