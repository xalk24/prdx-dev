package presentator

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("revision conflict")
)

type Deck struct {
	SchemaVersion string    `json:"schemaVersion"`
	DeckID        string    `json:"deckId"`
	Canvas        Canvas    `json:"canvas"`
	Theme         Theme     `json:"theme"`
	Slides        []Slide   `json:"slides"`
	Comments      []Comment `json:"comments"`
}

type Canvas struct{ Width, Height int }

func (c Canvas) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}{c.Width, c.Height})
}
func (c *Canvas) UnmarshalJSON(b []byte) error {
	var v struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	c.Width, c.Height = v.Width, v.Height
	return nil
}

type Theme struct {
	Version    string            `json:"version"`
	Palette    map[string]string `json:"palette"`
	Typography Typography        `json:"typography"`
	Background string            `json:"background"`
}
type Typography struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}
type Slide struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Elements []json.RawMessage `json:"elements"`
}
type Comment struct {
	ID       string  `json:"id"`
	SlideID  string  `json:"slideId"`
	Anchor   *Anchor `json:"anchor,omitempty"`
	Text     string  `json:"text"`
	Resolved bool    `json:"resolved"`
}
type Anchor struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func (d Deck) Validate() error {
	if d.SchemaVersion != "1.0" {
		return fmt.Errorf("unsupported schemaVersion %q", d.SchemaVersion)
	}
	if d.DeckID == "" || d.Canvas.Width != 1920 || d.Canvas.Height != 1080 {
		return errors.New("invalid deck identity or canvas")
	}
	if len(d.Slides) < 1 || len(d.Slides) > 30 {
		return errors.New("slides must contain 1..30 items")
	}
	seen := map[string]bool{}
	for _, s := range d.Slides {
		if s.ID == "" || seen[s.ID] || len(s.Elements) > 100 {
			return errors.New("invalid or duplicate slide id, or element limit exceeded")
		}
		seen[s.ID] = true
		for _, raw := range s.Elements {
			var h struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(raw, &h) != nil || (h.Type != "text" && h.Type != "image" && h.Type != "shape") {
				return errors.New("unsupported element")
			}
		}
	}
	for _, c := range d.Comments {
		if !seen[c.SlideID] || len(c.Text) > 2000 {
			return errors.New("invalid comment")
		}
	}
	return nil
}

type Project struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Revision int    `json:"revision"`
	Deck     Deck   `json:"deck"`
}
type Job struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"projectId"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	BaseRevision int       `json:"baseRevision"`
	Candidate    *Deck     `json:"candidate,omitempty"`
	Artifact     []byte    `json:"-"`
	Error        *JobError `json:"error,omitempty"`
}
type JobError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type GenerationRequest struct {
	Brief   string `json:"brief"`
	Locale  string `json:"locale"`
	Profile string `json:"profile"`
}
type PredictorXPort interface {
	Generate(ctx Context, request GenerationRequest, base Deck) (Deck, error)
}
type PDFPort interface {
	Render(ctx Context, deck Deck) ([]byte, error)
}
type Context interface {
	Done() <-chan struct{}
	Err() error
}
