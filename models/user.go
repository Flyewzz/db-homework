package models


import (
	strfmt "github.com/go-openapi/strfmt"
)

type User struct {
	About string `json:"about,omitempty"`
	Email strfmt.Email `json:"email"`
	Fullname string `json:"fullname"`
	Nickname string `json:"nickname,omitempty"`
}
