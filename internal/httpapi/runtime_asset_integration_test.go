//go:build integration

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xalk24/prdx-dev/internal/httpapi"
	"github.com/xalk24/prdx-dev/internal/presentator"
)

func TestRuntimeAssetExportAndMissingAssetFailure(t *testing.T) {
	store := presentator.NewStore()
	renderer := presentator.ChromiumPDF{NodePath: "node", ScriptPath: "../../scripts/render-pdf.mjs"}
	service := presentator.NewService(store, presentator.FixturePredictorX{}, renderer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.Run(ctx)
	server := httptest.NewServer(httpapi.New(store, service))
	defer server.Close()

	var project struct {
		ID       string `json:"id"`
		Revision int    `json:"revision"`
	}
	request(t, http.MethodPost, server.URL+"/api/v1/projects", `{"title":"runtime asset"}`, nil, http.StatusCreated, &project)
	assetID := uploadPNG(t, server.URL, project.ID)
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+project.ID+"/assets/"+assetID, "", nil, http.StatusOK, nil)
	var foreignProject struct {
		ID string `json:"id"`
	}
	request(t, http.MethodPost, server.URL+"/api/v1/projects", `{"title":"foreign project"}`, nil, http.StatusCreated, &foreignProject)
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+foreignProject.ID+"/assets/"+assetID, "", nil, http.StatusNotFound, nil)

	var snapshot struct {
		Deck     presentator.Deck `json:"deck"`
		Revision int              `json:"revision"`
	}
	request(t, http.MethodGet, server.URL+"/api/v1/projects/"+project.ID+"/deck", "", nil, http.StatusOK, &snapshot)
	snapshot.Deck.Slides[0].Elements = []json.RawMessage{json.RawMessage(fmt.Sprintf(`{"id":"image","type":"image","frame":{"x":80,"y":80,"width":800,"height":600},"visible":true,"locked":false,"assetId":%q,"fit":"cover"}`, assetID))}
	saveDeck(t, server.URL, project.ID, snapshot.Revision, snapshot.Deck, &snapshot)

	job := exportAndWait(t, server.URL, project.ID, snapshot.Revision, "asset-success")
	if job.Status != "succeeded" {
		t.Fatalf("asset export status=%s error=%+v", job.Status, job.Error)
	}
	response, err := http.Get(server.URL + "/api/v1/export-jobs/" + job.ID + "/download")
	if err != nil {
		t.Fatal(err)
	}
	pdf, readErr := io.ReadAll(response.Body)
	response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK || !bytes.Contains(pdf, []byte("/Subtype /Image")) {
		t.Fatalf("runtime asset not embedded: status=%d bytes=%d readErr=%v", response.StatusCode, len(pdf), readErr)
	}

	snapshot.Deck.Slides[0].Elements = []json.RawMessage{json.RawMessage(`{"id":"image","type":"image","frame":{"x":0,"y":0,"width":100,"height":100},"visible":true,"locked":false,"assetId":"missing","fit":"cover"}`)}
	saveDeck(t, server.URL, project.ID, snapshot.Revision, snapshot.Deck, &snapshot)
	job = exportAndWait(t, server.URL, project.ID, snapshot.Revision, "asset-missing")
	if job.Status != "failed" || job.Error == nil || job.Error.Code != "asset_missing" || len(job.Artifact) != 0 {
		t.Fatalf("missing asset did not fail closed: %+v", job)
	}
	request(t, http.MethodGet, server.URL+"/api/v1/export-jobs/"+job.ID+"/download", "", nil, http.StatusConflict, nil)
}

func uploadPNG(t *testing.T, baseURL, projectID string) string {
	t.Helper()
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
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/projects/"+projectID+"/assets", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload status=%d", resp.StatusCode)
	}
	var asset struct {
		ID string `json:"id"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		t.Fatal(err)
	}
	return asset.ID
}

func saveDeck(t *testing.T, baseURL, projectID string, revision int, deck presentator.Deck, snapshot any) {
	t.Helper()
	body, err := json.Marshal(deck)
	if err != nil {
		t.Fatal(err)
	}
	request(t, http.MethodPut, baseURL+"/api/v1/projects/"+projectID+"/deck", string(body), map[string]string{"If-Match": fmt.Sprintf("\"%d\"", revision)}, http.StatusOK, snapshot)
}

func exportAndWait(t *testing.T, baseURL, projectID string, revision int, key string) presentator.Job {
	t.Helper()
	var job presentator.Job
	request(t, http.MethodPost, baseURL+"/api/v1/projects/"+projectID+"/export-jobs", fmt.Sprintf(`{"revision":%d}`, revision), map[string]string{"Idempotency-Key": key}, http.StatusAccepted, &job)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		request(t, http.MethodGet, baseURL+"/api/v1/export-jobs/"+job.ID, "", nil, http.StatusOK, &job)
		if job.Status == "succeeded" || job.Status == "failed" {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("export %s timed out", job.ID)
	return job
}
