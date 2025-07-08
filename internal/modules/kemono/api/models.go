package api

import (
	"encoding/json"
)

type Root struct {
	Properties  *Properties     `json:"props"`
	Results     []Result        `json:"results"`
	Attachments *[][]Attachment `json:"result_attachments"`
	Previews    *[][]Thumbnail  `json:"result_previews"`
}

type Properties struct {
	CurrentPage string      `json:"currentPage"`
	ID          string      `json:"id"`
	Service     string      `json:"service"`
	Name        string      `json:"name"`
	Count       json.Number `json:"count"`
	Limit       json.Number `json:"limit"`
}

type Result struct {
	ID        string     `json:"id"`
	User      string     `json:"user"`
	Service   string     `json:"service"`
	Title     string     `json:"title"`
	Published CustomTime `json:"published"`
}

type PostRoot struct {
	Post        Post         `json:"post"`
	Attachments []Attachment `json:"attachments"`
	Previews    []Thumbnail  `json:"previews"`
	Videos      []Video      `json:"videos"` // Placeholder if videos are not defined
}

type Post struct {
	ID          string       `json:"id"`
	User        string       `json:"user"`
	Title       string       `json:"title"`
	Embed       Embed        `json:"embed"`
	Content     string       `json:"content"`
	File        File         `json:"file"`
	Attachments []Attachment `json:"attachments"`
	Published   CustomTime   `json:"published"`
}

type Embed struct {
	Url         string  `json:"url"`
	Subject     *string `json:"subject"`
	Description *string `json:"description"`
}

type File struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Attachment struct {
	Name   string  `json:"name"`
	Path   string  `json:"path"`
	Server *string `json:"server"`
}

type Video struct {
	Name      string  `json:"name"`
	Path      string  `json:"path"`
	Extension string  `json:"extension"`
	Server    *string `json:"server"`
}

type Thumbnail struct {
	Type   string  `json:"type"`
	Server *string `json:"server"`
	Name   string  `json:"name"`
	Path   string  `json:"path"`
}

type Comment struct {
	ID            string     `json:"id"`
	ParentID      *string    `json:"parent_id"`
	Commenter     string     `json:"commenter"`
	CommenterName string     `json:"commenter_name"`
	Content       string     `json:"content"`
	Published     CustomTime `json:"published"`
	Revisions     []Revision `json:"revisions"`
}

type Revision struct {
	ID      int        `json:"id"`
	Content string     `json:"content"`
	Added   CustomTime `json:"added"`
}
