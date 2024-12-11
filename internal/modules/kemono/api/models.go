package api

import "time"

type Root struct {
	Results []Result `json:"results"`
}

type Result struct {
	ID        string     `json:"id"`
	User      string     `json:"user"`
	Service   string     `json:"service"`
	Title     string     `json:"title"`
	Published CustomTime `json:"published"`
}

type PostRoot struct {
	Post        Post          `json:"post"`
	Attachments []Attachment  `json:"attachments"`
	Previews    []Thumbnail   `json:"previews"`
	Videos      []interface{} `json:"videos"` // Placeholder if videos are not defined
}

type Post struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	File      File       `json:"file"`
	Published CustomTime `json:"published"`
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
	Published     time.Time  `json:"published"`
	Revisions     []Revision `json:"revisions"`
}

type Revision struct {
	ID      int       `json:"id"`
	Content string    `json:"content"`
	Added   time.Time `json:"added"`
}
