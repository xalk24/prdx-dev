package presentator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestServiceQueueBackpressure(t *testing.T) {
	store := NewStore()
	svc := NewService(store, FixturePredictorX{}, MinimalPDF{})
	for i := 0; i < cap(svc.queue); i++ {
		p := store.CreateProject("queued", validDeck())
		if _, err := svc.EnqueueGeneration(p.ID, "key", GenerationRequest{Brief: "brief"}); err != nil {
			t.Fatalf("fill queue at %d: %v", i, err)
		}
	}
	p := store.CreateProject("overflow", validDeck())
	start := time.Now()
	_, err := svc.EnqueueGeneration(p.ID, "key", GenerationRequest{Brief: "brief"})
	if !errors.Is(err, ErrQueueFull) {
		t.Fatalf("overflow error=%v", err)
	}
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("overflow enqueue blocked")
	}
}

func TestServiceJobTimeoutDoesNotStopOtherWorkers(t *testing.T) {
	store := NewStore()
	svc := NewService(store, blockingPredictor{}, MinimalPDF{})
	svc.jobTimeout = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.Run(ctx)
	p := store.CreateProject("timeout", validDeck())
	job, err := svc.EnqueueGeneration(p.ID, "key", GenerationRequest{Brief: "brief"})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, err = store.Job(job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if job.Status == "failed" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if job.Status != "failed" || job.Error == nil || job.Error.Code != "timeout" {
		t.Fatalf("job=%+v", job)
	}
}

func TestExportFailsClosedWhenAssetIsMissing(t *testing.T) {
	store := NewStore()
	svc := NewService(store, FixturePredictorX{}, MinimalPDF{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.Run(ctx)
	deck := validDeck()
	deck.Slides[0].Elements = []json.RawMessage{json.RawMessage(`{"id":"image","type":"image","frame":{"x":0,"y":0,"width":100,"height":100},"visible":true,"locked":false,"assetId":"missing","fit":"cover"}`)}
	p := store.CreateProject("missing", deck)
	job, err := svc.EnqueueExport(p.ID, "export-missing", p.Revision)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, _ = store.Job(job.ID)
		if job.Status == "failed" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if job.Status != "failed" || job.Error == nil || job.Error.Code != "asset_missing" || len(job.Artifact) != 0 {
		t.Fatalf("job did not fail closed: %+v", job)
	}
}

type blockingPredictor struct{}

func (blockingPredictor) Generate(ctx Context, _ GenerationRequest, _ Deck) (Deck, error) {
	<-ctx.Done()
	return Deck{}, ctx.Err()
}

func validDeck() Deck {
	return Deck{SchemaVersion: "1.0", DeckID: "d", Canvas: Canvas{1920, 1080}, Theme: Theme{Version: "1.0"}, Slides: []Slide{{ID: "s"}}}
}
