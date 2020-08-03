package awsapi

import (
	"encoding/json"
)

type from struct {
	Id int `json:"id"`
}

type file struct {
	FileId   string `json:"file_id"`
	FileSize int    `json:"file_size"`
}

type voice struct {
	Duration int    `json:"duration"`
	MimeType string `json:"mime_type"`
	file
}

type photo struct {
	file
	Width  int `json:"width"`
	Height int `json:"height"`
}

type TGUserMsg struct {
	From  from    `json:"from"`
	Voice voice   `json:"voice"`
	Date  int     `json:"date"`
	Photo []photo `json:"photo"`
	Text  string  `json:"text"`
}

func NewTgUserMsg(orig string) (*TGUserMsg, error) {
	msg := &TGUserMsg{}
	err := json.Unmarshal([]byte(orig), msg)
	return msg, err
}

func (um *TGUserMsg) TGID() string {
	return string(um.From.Id)
}