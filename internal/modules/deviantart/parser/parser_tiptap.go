package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type Document struct {
	Version  json.Number `json:"version"`
	Document Content     `json:"document"`
}

type Content struct {
	Type    string    `json:"type"`
	Content []Element `json:"content"`
}

type Element struct {
	Type  string `json:"type"`
	Attrs struct {
		Indentation FlexibleString `json:"indentation"`
		TextAlign   string         `json:"textAlign"`
	} `json:"attrs"`
	Content []Text `json:"content"`
}

type Text struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Marks []Mark `json:"marks"`
}

type Mark struct {
	Type string `json:"type"`
}

type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleString(s)
		return nil
	}

	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*fs = FlexibleString(fmt.Sprintf("%g", n))
		return nil
	}

	return fmt.Errorf("unable to unmarshal FlexibleString: %s", string(data))
}

// ParseTipTapFormat parses a JSON string in TipTap format and returns the HTML representation
func ParseTipTapFormat(jsonStr string) (string, error) {
	var doc Document
	err := json.Unmarshal([]byte(jsonStr), &doc)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer

	for _, element := range doc.Document.Content {
		if element.Type == "paragraph" {
			indentation := strings.Repeat("&nbsp;", 4*toInt(string(element.Attrs.Indentation)))
			textAlign := element.Attrs.TextAlign
			if textAlign == "" {
				textAlign = "left"
			}

			buffer.WriteString(fmt.Sprintf(`<p style="text-align: %s;">%s`, textAlign, indentation))

			for _, content := range element.Content {
				if content.Type == "text" {
					text := content.Text
					for _, mark := range content.Marks {
						if mark.Type == "bold" {
							text = fmt.Sprintf("<strong>%s</strong>", text)
						}
						if mark.Type == "italic" {
							text = fmt.Sprintf("<em>%s</em>", text)
						}
					}
					buffer.WriteString(text)
				} else if content.Type == "hardBreak" {
					buffer.WriteString("<br />")
				}
			}

			buffer.WriteString("</p>\n")
		}
	}

	return buffer.String(), nil
}

// Helper function to convert strings to integers
func toInt(str string) int {
	var i int
	_, err := fmt.Sscanf(str, "%d", &i)
	if err != nil {
		return 0
	}
	return i
}
