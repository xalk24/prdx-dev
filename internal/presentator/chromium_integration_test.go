//go:build integration

package presentator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	const golden = "259794d1d877e7cec788c0d0cfa6efb38f82b3fa6d7251c1dfa67e0697e4817e"
	if got := hex.EncodeToString(sum[:]); got != golden {
		t.Fatalf("renderer fidelity changed: got %s want %s", got, golden)
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
	pdf, err := renderer.Render(ctx, deck)
	if err != nil {
		t.Fatal(err)
	}
	if len(pdf) < 500 {
		t.Fatalf("PDF is unexpectedly small: %d", len(pdf))
	}
}
