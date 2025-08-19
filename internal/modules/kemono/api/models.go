package api

type Profile struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Service    string     `json:"service"`
	Indexed    CustomTime `json:"indexed"`
	Updated    CustomTime `json:"updated"`
	PublicID   string     `json:"public_id"`
	RelationID *string    `json:"relation_id"`
	HasChats   bool       `json:"has_chats"`
	PostCount  int        `json:"post_count"`
	DmCount    int        `json:"dm_count"`
	ShareCount int        `json:"share_count"`
	ChatCount  int        `json:"chat_count"`
}

type QuickPost struct {
	ID          string     `json:"id"`
	User        string     `json:"user"`
	Service     string     `json:"service"`
	Title       string     `json:"title"`
	Substring   string     `json:"substring"`
	Published   CustomTime `json:"published"`
	File        File       `json:"file"`
	Attachments []File     `json:"attachments"`
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
