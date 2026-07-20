package presentator

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrConflict            = errors.New("revision conflict")
	ErrQueueFull           = errors.New("job queue full")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
	ErrAssetMissing        = errors.New("asset missing")
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

type Asset struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	MIME      string `json:"mime"`
	Size      int    `json:"size"`
	Data      []byte `json:"-"`
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
			if validateElement(raw) != nil {
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

func (d *Deck) Normalize() {
	if d.Slides == nil {
		d.Slides = []Slide{}
	}
	if d.Comments == nil {
		d.Comments = []Comment{}
	}
	if d.Theme.Palette == nil {
		d.Theme.Palette = map[string]string{}
	}
	for i := range d.Slides {
		if d.Slides[i].Elements == nil {
			d.Slides[i].Elements = []json.RawMessage{}
		}
	}
}

func validateElement(raw json.RawMessage) error {
	var value struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		AssetID string `json:"assetId"`
		Fit     string `json:"fit"`
		Crop    *struct {
			Version string  `json:"version"`
			X       float64 `json:"x"`
			Y       float64 `json:"y"`
			Width   float64 `json:"width"`
			Height  float64 `json:"height"`
		} `json:"crop"`
		Shape       string  `json:"shape"`
		Stroke      string  `json:"stroke"`
		StrokeWidth float64 `json:"strokeWidth"`
		Opacity     float64 `json:"opacity"`
		Frame       struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"frame"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	if value.ID == "" || !finite(value.Frame.X, value.Frame.Y, value.Frame.Width, value.Frame.Height) || value.Frame.Width <= 0 || value.Frame.Height <= 0 {
		return errors.New("invalid element")
	}
	switch value.Type {
	case "text":
		return nil
	case "image":
		if value.AssetID == "" || (value.Fit != "contain" && value.Fit != "cover") {
			return errors.New("invalid image element")
		}
		if value.Crop != nil {
			crop := value.Crop
			if crop.Version != "1.0" || !finite(crop.X, crop.Y, crop.Width, crop.Height) || crop.X < 0 || crop.Y < 0 || crop.Width <= 0 || crop.Height <= 0 || crop.X+crop.Width > 1 || crop.Y+crop.Height > 1 {
				return errors.New("invalid image crop")
			}
		}
		return nil
	case "shape":
		if value.Shape != "rect" && value.Shape != "ellipse" && value.Shape != "line" {
			return errors.New("invalid shape element")
		}
		if value.Stroke == "" || !finite(value.StrokeWidth, value.Opacity) || value.StrokeWidth < 0 || value.Opacity < 0 || value.Opacity > 1 {
			return errors.New("invalid shape style")
		}
		return nil
	default:
		return errors.New("unsupported element")
	}
}

func finite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
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
	Render(ctx Context, deck Deck, assets map[string]string) ([]byte, error)
}
type Context interface {
	Done() <-chan struct{}
	Err() error
}
