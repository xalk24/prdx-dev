package presentator

import (
	"context"
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
