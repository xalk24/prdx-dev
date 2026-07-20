package presentator

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMinimalPDF(t *testing.T) {
	deck := Deck{Slides: []Slide{{ID: "s1", Name: "First (deck)"}, {ID: "s2", Name: "Second"}}}
	got, err := (MinimalPDF{}).Render(context.Background(), deck, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), "%PDF-1.4") || !strings.HasSuffix(string(got), "%%EOF\n") {
		t.Fatal("invalid PDF envelope")
	}
	pdf := string(got)
	if !strings.Contains(pdf, "/Count 2") || !strings.Contains(pdf, "First \\(deck\\)") {
		t.Fatal("PDF page count or escaped content is invalid")
	}
	if strings.Index(pdf, "First") >= strings.Index(pdf, "Second") {
		t.Fatal("PDF slide order is invalid")
	}
}

func TestDeckValidateImageCropAndLineStyle(t *testing.T) {
	tests := []struct {
		name    string
		element string
		wantErr bool
	}{
		{"valid crop", `{"id":"image","type":"image","frame":{"x":0,"y":0,"width":100,"height":100},"visible":true,"locked":false,"assetId":"asset-1","fit":"cover","crop":{"version":"1.0","x":0.1,"y":0.2,"width":0.7,"height":0.6}}`, false},
		{"crop outside source", `{"id":"image","type":"image","frame":{"x":0,"y":0,"width":100,"height":100},"visible":true,"locked":false,"assetId":"asset-1","fit":"cover","crop":{"version":"1.0","x":0.5,"y":0,"width":0.6,"height":1}}`, true},
		{"unknown crop version", `{"id":"image","type":"image","frame":{"x":0,"y":0,"width":100,"height":100},"visible":true,"locked":false,"assetId":"asset-1","fit":"cover","crop":{"version":"2.0","x":0,"y":0,"width":1,"height":1}}`, true},
		{"custom line", `{"id":"line","type":"shape","frame":{"x":0,"y":0,"width":100,"height":2},"visible":true,"locked":false,"shape":"line","fill":"transparent","stroke":"#123456","strokeWidth":7,"opacity":0.8}`, false},
		{"negative line width", `{"id":"line","type":"shape","frame":{"x":0,"y":0,"width":100,"height":2},"visible":true,"locked":false,"shape":"line","fill":"transparent","stroke":"#123456","strokeWidth":-1,"opacity":1}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deck := validDeck()
			deck.Slides[0].Elements = []json.RawMessage{json.RawMessage(tt.element)}
			if err := deck.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
func TestDeckValidate(t *testing.T) {
	tests := []struct {
		name string
		deck Deck
		ok   bool
	}{{"valid", Deck{SchemaVersion: "1.0", DeckID: "d", Canvas: Canvas{1920, 1080}, Slides: []Slide{{ID: "s"}}}, true}, {"unknown version", Deck{SchemaVersion: "2", DeckID: "d", Canvas: Canvas{1920, 1080}, Slides: []Slide{{ID: "s"}}}, false}, {"duplicate slide", Deck{SchemaVersion: "1.0", DeckID: "d", Canvas: Canvas{1920, 1080}, Slides: []Slide{{ID: "s"}, {ID: "s"}}}, false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if (tt.deck.Validate() == nil) != tt.ok {
				t.Fatalf("Validate ok mismatch")
			}
		})
	}
}
