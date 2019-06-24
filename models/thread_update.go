package models

type ThreadUpdate struct {
	Message string `json:"message,omitempty"`
	Title   string `json:"title,omitempty"`
}
