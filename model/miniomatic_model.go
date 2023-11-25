package model

type Credentials struct {
	RandNum      string
	RootUser     string
	RootPassword string
}
type Item struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

var Items []Item

type Post struct {
	Storage string `json:"storage"`
	Bucket  string `json:"bucket"`
}

type Resp struct {
	Status    string `json:"status,omitempty"`
	ID        string `json:"id,omitempty"`
	Storage   string `json:"storage,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	URL       string `json:"url,omitempty"`
	AccessKey string `json:"accesskey,omitempty"`
	SecretKey string `json:"secretkey,omitempty"`
}

type Record struct {
	Status     string `json:"status,omitempty"`
	Date       string `json:"date,omitempty"`
	ID         string `json:"id,omitempty"`
	InitBucket string `json:"initbucket,omitempty"`
	URL        string `json:"url,omitempty"`
	Storage    string `json:"storage,omitempty"`
}
