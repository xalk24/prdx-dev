//go:build integration

package presentator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestChromiumRendererGolden(t *testing.T) {
	root := filepath.Join("..", "..")
	build := exec.Command("npm", "run", "build:renderer")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build renderer: %v: %s", err, output)
	}
	fixture, err := os.Open(filepath.Join(root, "api", "fixtures", "fidelity-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", filepath.Join("scripts", "render-pdf.mjs"), "--screenshot")
	cmd.Dir, cmd.Stdin = root, fixture
	png, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(png)
	const golden = "7aebbb548879b296527266576a17ceab39cd6a2729c610a26ad0ae726ea6802c"
	if got := hex.EncodeToString(sum[:]); got != golden {
		t.Fatalf("renderer fidelity changed: got %s want %s", got, golden)
	}
}

func TestChromiumRendererProductionFidelity(t *testing.T) {
	root := filepath.Join("..", "..")
	fixture, err := os.Open(filepath.Join(root, "api", "fixtures", "fidelity-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", filepath.Join("scripts", "render-pdf.mjs"), "--inspect")
	cmd.Dir, cmd.Stdin = root, fixture
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		FontFamily  string  `json:"fontFamily"`
		TextHeight  float64 `json:"textHeight"`
		LineHeight  float64 `json:"lineHeight"`
		LineColor   string  `json:"lineColor"`
		LineWidth   string  `json:"lineWidth"`
		CropVersion string  `json:"cropVersion"`
		CropLeft    string  `json:"cropLeft"`
		CropWidth   string  `json:"cropWidth"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	if got.FontFamily != "Inter" || got.TextHeight <= got.LineHeight {
		t.Fatalf("bundled Inter did not produce wrapped text: %+v", got)
	}
	if got.LineColor != "rgb(220, 38, 38)" || got.LineWidth != "5px" {
		t.Fatalf("custom line style lost: %+v", got)
	}
	if got.CropVersion != "1.0" || got.CropLeft == "0px" || got.CropWidth == "360px" {
		t.Fatalf("non-default crop geometry lost: %+v", got)
	}
}

func TestChromiumPDFReadinessAndArtifact(t *testing.T) {
	renderer := ChromiumPDF{NodePath: "node", ScriptPath: "../../scripts/render-pdf.mjs"}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := renderer.Ready(ctx); err != nil {
		t.Fatal(err)
	}
	deck := validDeck()
	pdf, err := renderer.Render(ctx, deck, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 500 {
		t.Fatalf("PDF is unexpectedly small: %d", len(pdf))
	}
}

func TestRuntimeAssetIsEmbeddedInPDF(t *testing.T) {
	store := NewStore()
	deck := validDeck()
	project := store.CreateProject("asset PDF", deck)
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	asset, err := store.CreateAsset(project.ID, "image/png", png)
	if err != nil {
		t.Fatal(err)
	}
	deck.DeckID = project.ID
	deck.Slides[0].Elements = []json.RawMessage{json.RawMessage(`{"id":"image","type":"image","frame":{"x":80,"y":80,"width":800,"height":600},"visible":true,"locked":false,"assetId":"` + asset.ID + `","fit":"cover"}`)}
	assets, err := store.RendererAssets(project.ID, deck)
	if err != nil {
		t.Fatal(err)
	}
	renderer := ChromiumPDF{NodePath: "node", ScriptPath: "../../scripts/render-pdf.mjs"}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pdf, err := renderer.Render(ctx, deck, assets)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Image")) {
		t.Fatal("PDF does not contain an embedded image object")
	}
}
