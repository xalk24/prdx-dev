package presentator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// FixturePredictorX is deterministic local development behavior behind the production port.
type FixturePredictorX struct{}

func (FixturePredictorX) Generate(ctx Context, req GenerationRequest, base Deck) (Deck, error) {
	select {
	case <-ctx.Done():
		return Deck{}, ctx.Err()
	default:
	}
	if strings.TrimSpace(req.Brief) == "" {
		return Deck{}, fmt.Errorf("invalid input")
	}
	d := base
	d.Comments = nil
	d.Normalize()
	return d, nil
}

// MinimalPDF is a dependency-free gate renderer. Production replaces this port with Chromium.
type MinimalPDF struct{}

func (MinimalPDF) Render(ctx Context, deck Deck, _ map[string]string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	write := func(s string) { offsets = append(offsets, b.Len()); b.WriteString(s) }
	n := len(deck.Slides)
	kids := make([]string, n)
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", 3+i*2)
	}
	write("1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n")
	write(fmt.Sprintf("2 0 obj << /Type /Pages /Count %d /Kids [%s] >> endobj\n", n, strings.Join(kids, " ")))
	for i, s := range deck.Slides {
		page := 3 + i*2
		content := page + 1
		stream := fmt.Sprintf("BT /F1 24 Tf 72 500 Td (%s) Tj ET", pdfEscape(s.Name))
		write(fmt.Sprintf("%d 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 960 540] /Resources << /Font << /F1 << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> >> >> /Contents %d 0 R >> endobj\n", page, content))
		write(fmt.Sprintf("%d 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", content, len(stream), stream))
	}
	xref := b.Len()
	b.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(offsets)))
	for _, o := range offsets[1:] {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", o))
	}
	b.WriteString(fmt.Sprintf("trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref))
	return b.Bytes(), nil
}
func pdfEscape(s string) string {
	return strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)").Replace(s)
}

type ChromiumPDF struct {
	NodePath   string
	ScriptPath string
}

func (c ChromiumPDF) Ready(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.NodePath, c.ScriptPath, "--ready")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("chromium readiness: %w: %s", err, output)
	}
	var readiness struct {
		Ready             bool   `json:"ready"`
		RendererVersion   string `json:"rendererVersion"`
		PlaywrightVersion string `json:"playwrightVersion"`
		ChromiumVersion   string `json:"chromiumVersion"`
	}
	if err := json.Unmarshal(output, &readiness); err != nil {
		return fmt.Errorf("decode chromium readiness: %w", err)
	}
	if !readiness.Ready || readiness.RendererVersion != "1.1.0" || readiness.PlaywrightVersion != "1.60.0" || readiness.ChromiumVersion == "" {
		return fmt.Errorf("unexpected chromium readiness: %+v", readiness)
	}
	return nil
}

func (c ChromiumPDF) Render(ctx Context, deck Deck, assets map[string]string) ([]byte, error) {
	processContext, ok := ctx.(context.Context)
	if !ok {
		return nil, errors.New("chromium renderer requires context.Context")
	}
	deck.Normalize()
	payload, err := json.Marshal(struct {
		Deck   Deck              `json:"deck"`
		Assets map[string]string `json:"assets"`
	}{Deck: deck, Assets: assets})
	if err != nil {
		return nil, fmt.Errorf("encode renderer request: %w", err)
	}
	cmd := exec.CommandContext(processContext, c.NodePath, c.ScriptPath)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("chromium render: %w: %.2048s", err, stderr.String())
	}
	if !bytes.HasPrefix(stdout.Bytes(), []byte("%PDF-")) {
		return nil, errors.New("chromium renderer returned invalid PDF")
	}
	return stdout.Bytes(), nil
}

var _ context.Context
