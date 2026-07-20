package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xalk24/prdx-dev/internal/httpapi"
	"github.com/xalk24/prdx-dev/internal/presentator"
)

func TestVerticalSlice(t *testing.T) {
	store := presentator.NewStore()
	svc := presentator.NewService(store, presentator.FixturePredictorX{}, presentator.MinimalPDF{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.Run(ctx)
	server := httptest.NewServer(httpapi.New(store, svc))
	defer server.Close()
	var project struct {
		ID       string `json:"id"`
		Revision int    `json:"revision"`
	}
	request(t, http.MethodPost, server.URL+"/api/v1/projects", `{"title":"Demo"}`, nil, http.StatusCreated, &project)
	var snapshot struct {
		Deck     presentator.Deck `json:"deck"`
		Revision int              `json:"revision"`
	}
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+project.ID+"/deck", "", nil, http.StatusOK, &snapshot)
	saved, _ := json.Marshal(snapshot.Deck)
	request(t, http.MethodPut, server.URL+"/api/v1/projects/"+project.ID+"/deck", string(saved), map[string]string{"If-Match": "\"1\""}, http.StatusOK, &snapshot)
	var job presentator.Job
	request(t, http.MethodPost, server.URL+"/api/v1/projects/"+project.ID+"/generation-jobs", `{"brief":"Create a pitch"}`, map[string]string{"Idempotency-Key": "generation-1"}, http.StatusAccepted, &job)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		request(t, http.MethodGet, server.URL+"/api/v1/generation-jobs/"+job.ID, "", nil, http.StatusOK, &job)
		if job.Status == "succeeded" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if job.Status != "succeeded" {
		t.Fatalf("job status=%s", job.Status)
	}
	request(t, http.MethodPost, server.URL+"/api/v1/generation-jobs/"+job.ID+"/apply", `{"baseRevision":2}`, map[string]string{"Idempotency-Key": "apply-1"}, http.StatusOK, &snapshot)
	if snapshot.Revision != 3 {
		t.Fatalf("revision=%d", snapshot.Revision)
	}
}
func TestRevisionConflict(t *testing.T) {
	store := presentator.NewStore()
	svc := presentator.NewService(store, presentator.FixturePredictorX{}, presentator.MinimalPDF{})
	server := httptest.NewServer(httpapi.New(store, svc))
	defer server.Close()
	var p struct {
		ID string `json:"id"`
	}
	request(t, http.MethodPost, server.URL+"/api/v1/projects", `{"title":"Demo"}`, nil, 201, &p)
	var snap struct {
		Deck presentator.Deck `json:"deck"`
	}
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+p.ID+"/deck", "", nil, 200, &snap)
	b, _ := json.Marshal(snap.Deck)
	request(t, http.MethodPut, server.URL+"/api/v1/projects/"+p.ID+"/deck", string(b), map[string]string{"If-Match": "99"}, 409, nil)
}
func request(t *testing.T, method, url, body string, headers map[string]string, want int, out any) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("%s %s status=%d want=%d", method, url, resp.StatusCode, want)
	}
	if out != nil && json.NewDecoder(resp.Body).Decode(out) != nil {
		t.Fatal("decode response")
	}
}
