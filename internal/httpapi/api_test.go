package httpapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
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
	var candidate presentator.Deck
	request(t, http.MethodGet, server.URL+"/api/v1/generation-jobs/"+job.ID+"/candidate", "", nil, http.StatusOK, &candidate)
	if candidate.Comments == nil || candidate.Slides == nil || candidate.Slides[0].Elements == nil {
		t.Fatal("candidate arrays were not normalized")
	}
	request(t, http.MethodPost, server.URL+"/api/v1/generation-jobs/"+job.ID+"/apply", `{"baseRevision":2}`, map[string]string{"Idempotency-Key": "apply-1"}, http.StatusOK, &snapshot)
	if snapshot.Revision != 3 {
		t.Fatalf("revision=%d", snapshot.Revision)
	}
	request(t, http.MethodPost, server.URL+"/api/v1/generation-jobs/"+job.ID+"/apply", `{"baseRevision":2}`, map[string]string{"Idempotency-Key": "apply-1"}, http.StatusOK, &snapshot)
	if snapshot.Revision != 3 {
		t.Fatalf("replay revision=%d", snapshot.Revision)
	}
	request(t, http.MethodPost, server.URL+"/api/v1/generation-jobs/"+job.ID+"/apply", `{"baseRevision":3}`, map[string]string{"Idempotency-Key": "apply-1"}, http.StatusConflict, nil)
	var export presentator.Job
	request(t, http.MethodPost, server.URL+"/api/v1/projects/"+project.ID+"/export-jobs", `{"revision":3}`, map[string]string{"Idempotency-Key": "export-1"}, http.StatusAccepted, &export)
	for time.Now().Before(deadline.Add(time.Second)) {
		request(t, http.MethodGet, server.URL+"/api/v1/export-jobs/"+export.ID, "", nil, http.StatusOK, &export)
		if export.Status == "succeeded" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	resp, err := http.Get(server.URL + "/api/v1/export-jobs/" + export.ID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/pdf" {
		t.Fatalf("download status=%d content-type=%q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var artifact bytes.Buffer
	if _, err := artifact.ReadFrom(resp.Body); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(artifact.Bytes(), []byte("%PDF-1.4")) {
		t.Fatal("download is not a PDF")
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

func TestStrictRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		headers map[string]string
	}{
		{name: "title limit", body: `{"title":"` + string(bytes.Repeat([]byte("a"), 201)) + `"}`},
		{name: "second JSON value", body: `{"title":"ok"} {"title":"extra"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := presentator.NewStore()
			svc := presentator.NewService(store, presentator.FixturePredictorX{}, presentator.MinimalPDF{})
			server := httptest.NewServer(httpapi.New(store, svc))
			defer server.Close()
			request(t, http.MethodPost, server.URL+"/api/v1/projects", tt.body, tt.headers, http.StatusBadRequest, nil)
		})
	}
}

func TestApplyRequiresIdempotencyKey(t *testing.T) {
	store := presentator.NewStore()
	svc := presentator.NewService(store, presentator.FixturePredictorX{}, presentator.MinimalPDF{})
	server := httptest.NewServer(httpapi.New(store, svc))
	defer server.Close()
	request(t, http.MethodPost, server.URL+"/api/v1/generation-jobs/missing/apply", `{"baseRevision":1}`, nil, http.StatusBadRequest, nil)
}

func TestProjectAssetOwnershipAndMIME(t *testing.T) {
	store := presentator.NewStore()
	svc := presentator.NewService(store, presentator.FixturePredictorX{}, presentator.MinimalPDF{})
	server := httptest.NewServer(httpapi.New(store, svc))
	defer server.Close()
	var first, second struct {
		ID string `json:"id"`
	}
	request(t, http.MethodPost, server.URL+"/api/v1/projects", `{"title":"first"}`, nil, 201, &first)
	request(t, http.MethodPost, server.URL+"/api/v1/projects", `{"title":"second"}`, nil, 201, &second)
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "pixel.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(png); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/projects/"+first.ID+"/assets", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("upload status=%d", resp.StatusCode)
	}
	var asset struct{ ID, MIME string }
	if err = json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		t.Fatal(err)
	}
	if asset.MIME != "image/png" {
		t.Fatalf("mime=%s", asset.MIME)
	}
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+first.ID+"/assets/"+asset.ID, "", nil, 200, nil)
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+second.ID+"/assets/"+asset.ID, "", nil, 404, nil)
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
