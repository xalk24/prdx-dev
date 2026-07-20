package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/xalk24/prdx-dev/internal/presentator"
)

type API struct {
	store   *presentator.Store
	service *presentator.Service
}

func New(store *presentator.Store, service *presentator.Service) http.Handler {
	a := &API{store: store, service: service}
	m := http.NewServeMux()
	m.HandleFunc("GET /api/v1/healthz", a.health)
	m.HandleFunc("POST /api/v1/projects", a.createProject)
	m.HandleFunc("GET /api/v1/projects/{projectId}/deck", a.getDeck)
	m.HandleFunc("PUT /api/v1/projects/{projectId}/deck", a.saveDeck)
	m.HandleFunc("POST /api/v1/projects/{projectId}/generation-jobs", a.createGeneration)
	m.HandleFunc("GET /api/v1/generation-jobs/{jobId}", a.getJob)
	m.HandleFunc("GET /api/v1/generation-jobs/{jobId}/candidate", a.candidate)
	m.HandleFunc("POST /api/v1/generation-jobs/{jobId}/apply", a.apply)
	m.HandleFunc("POST /api/v1/projects/{projectId}/assets", a.createAsset)
	m.HandleFunc("GET /api/v1/projects/{projectId}/assets/{assetId}", a.getAsset)
	m.HandleFunc("POST /api/v1/projects/{projectId}/export-jobs", a.createExport)
	m.HandleFunc("GET /api/v1/export-jobs/{jobId}", a.getExportJob)
	m.HandleFunc("GET /api/v1/export-jobs/{jobId}/download", a.downloadExport)
	return http.MaxBytesHandler(m, 11<<20)
}
func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	write(w, 200, map[string]string{"status": "ok"})
}
func (a *API) createProject(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Title string `json:"title"`
	}
	if decode(r, &v) != nil || strings.TrimSpace(v.Title) == "" || len(v.Title) > 200 {
		problem(w, r, 400, "invalid_input", "title is required", false)
		return
	}
	d := presentator.Deck{SchemaVersion: "1.0", Canvas: presentator.Canvas{Width: 1920, Height: 1080}, Theme: presentator.Theme{Version: "1.0", Palette: map[string]string{"primary": "#2563EB"}, Typography: presentator.Typography{Heading: "Inter", Body: "Inter"}, Background: "#FFFFFF"}, Slides: []presentator.Slide{{ID: "slide-1", Name: "Untitled", Elements: []json.RawMessage{}}}, Comments: []presentator.Comment{}}
	p := a.store.CreateProject(v.Title, d)
	write(w, 201, map[string]any{"id": p.ID, "title": p.Title, "revision": p.Revision})
}
func (a *API) getDeck(w http.ResponseWriter, r *http.Request) {
	p, err := a.store.Project(r.PathValue("projectId"))
	if err != nil {
		problemFor(w, r, err)
		return
	}
	w.Header().Set("ETag", fmt.Sprintf("\"%d\"", p.Revision))
	write(w, 200, map[string]any{"deck": p.Deck, "revision": p.Revision})
}
func (a *API) saveDeck(w http.ResponseWriter, r *http.Request) {
	rev, err := etag(r.Header.Get("If-Match"))
	if err != nil {
		problem(w, r, 400, "invalid_if_match", "If-Match revision is required", false)
		return
	}
	var d presentator.Deck
	if decode(r, &d) != nil {
		problem(w, r, 400, "invalid_json", "invalid JSON", false)
		return
	}
	if err = d.Validate(); err != nil {
		problem(w, r, 422, "invalid_deck", err.Error(), false)
		return
	}
	p, err := a.store.SaveDeck(r.PathValue("projectId"), rev, d)
	if err != nil {
		problemFor(w, r, err)
		return
	}
	write(w, 200, map[string]any{"deck": p.Deck, "revision": p.Revision})
}
func (a *API) createGeneration(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || len(key) > 200 {
		problem(w, r, 400, "idempotency_key_required", "Idempotency-Key is required", false)
		return
	}
	var req presentator.GenerationRequest
	if decode(r, &req) != nil || len(req.Brief) > 20000 || strings.TrimSpace(req.Brief) == "" {
		problem(w, r, 400, "invalid_input", "brief must contain 1..20000 characters", false)
		return
	}
	j, err := a.service.EnqueueGeneration(r.PathValue("projectId"), key, req)
	if err != nil {
		problemFor(w, r, err)
		return
	}
	write(w, 202, j)
}
func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	j, err := a.store.Job(r.PathValue("jobId"))
	if err != nil {
		problemFor(w, r, err)
		return
	}
	write(w, 200, j)
}
func (a *API) candidate(w http.ResponseWriter, r *http.Request) {
	j, err := a.store.Job(r.PathValue("jobId"))
	if err != nil {
		problemFor(w, r, err)
		return
	}
	if j.Status != "succeeded" || j.Candidate == nil {
		problem(w, r, 409, "candidate_not_ready", "candidate is not ready", true)
		return
	}
	write(w, 200, j.Candidate)
}
func (a *API) apply(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || len(key) > 200 {
		problem(w, r, 400, "idempotency_key_required", "Idempotency-Key must contain 1..200 characters", false)
		return
	}
	var req struct {
		Base int `json:"baseRevision"`
	}
	if decode(r, &req) != nil {
		problem(w, r, 400, "invalid_input", "baseRevision is required", false)
		return
	}
	p, err := a.service.Apply(r.PathValue("jobId"), key, req.Base)
	if err != nil {
		problemFor(w, r, err)
		return
	}
	write(w, 200, map[string]any{"deck": p.Deck, "revision": p.Revision})
}
func (a *API) createExport(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	var req struct {
		Revision int `json:"revision"`
	}
	if key == "" || len(key) > 200 || decode(r, &req) != nil {
		problem(w, r, 400, "invalid_input", "idempotency key and revision are required", false)
		return
	}
	j, err := a.service.EnqueueExport(r.PathValue("projectId"), key, req.Revision)
	if err != nil {
		problemFor(w, r, err)
		return
	}
	write(w, 202, j)
}

const maxAssetBytes = 10 << 20

func (a *API) createAsset(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxAssetBytes); err != nil {
		problem(w, r, 400, "invalid_asset", "multipart image up to 10 MB is required", false)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		problem(w, r, 400, "invalid_asset", "file field is required", false)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxAssetBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxAssetBytes {
		problem(w, r, 413, "asset_too_large", "asset must contain 1..10485760 bytes", false)
		return
	}
	mime, err := validateImage(data)
	if err != nil {
		problem(w, r, 415, "unsupported_asset", err.Error(), false)
		return
	}
	asset, err := a.store.CreateAsset(r.PathValue("projectId"), mime, data)
	if err != nil {
		problemFor(w, r, err)
		return
	}
	write(w, 201, map[string]any{"id": asset.ID, "projectId": asset.ProjectID, "mime": asset.MIME, "size": asset.Size, "url": fmt.Sprintf("/api/v1/projects/%s/assets/%s", asset.ProjectID, asset.ID)})
}

func (a *API) getAsset(w http.ResponseWriter, r *http.Request) {
	asset, err := a.store.Asset(r.PathValue("projectId"), r.PathValue("assetId"))
	if err != nil {
		problemFor(w, r, err)
		return
	}
	w.Header().Set("Content-Type", asset.MIME)
	w.Header().Set("Content-Length", strconv.Itoa(asset.Size))
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(asset.Data)
}

func validateImage(data []byte) (string, error) {
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg":
		if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
			return "", errors.New("image cannot be decoded")
		}
		return mime, nil
	case "image/webp":
		if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			return "", errors.New("invalid WebP")
		}
		return mime, nil
	default:
		return "", errors.New("only PNG, JPEG and WebP assets are allowed")
	}
}
func (a *API) getExportJob(w http.ResponseWriter, r *http.Request) {
	j, err := a.store.Job(r.PathValue("jobId"))
	if err != nil {
		problemFor(w, r, err)
		return
	}
	if j.Type != "export" {
		problem(w, r, 404, "not_found", "resource not found", false)
		return
	}
	write(w, 200, j)
}
func (a *API) downloadExport(w http.ResponseWriter, r *http.Request) {
	j, err := a.store.Job(r.PathValue("jobId"))
	if err != nil {
		problemFor(w, r, err)
		return
	}
	if j.Type != "export" {
		problem(w, r, 404, "not_found", "resource not found", false)
		return
	}
	if j.Status != "succeeded" || len(j.Artifact) == 0 {
		problem(w, r, 409, "artifact_not_ready", "PDF artifact is not ready", true)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=presentator-%s.pdf", j.ID))
	w.Header().Set("Content-Length", strconv.Itoa(len(j.Artifact)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(j.Artifact)
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
func etag(s string) (int, error) { s = strings.Trim(s, "\""); return strconv.Atoi(s) }
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problemFor(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, presentator.ErrNotFound):
		problem(w, r, 404, "not_found", "resource not found", false)
	case errors.Is(err, presentator.ErrConflict):
		problem(w, r, 409, "conflict", "resource state conflicts with request", true)
	case errors.Is(err, presentator.ErrQueueFull):
		problem(w, r, 503, "queue_full", "job queue is full", true)
	case errors.Is(err, presentator.ErrIdempotencyConflict):
		problem(w, r, 409, "idempotency_conflict", "Idempotency-Key was already used with a different payload", false)
	default:
		problem(w, r, 500, "internal", "internal error", true)
	}
}
func problem(w http.ResponseWriter, r *http.Request, status int, code, msg string, retry bool) {
	cid := r.Header.Get("X-Correlation-ID")
	if cid == "" {
		cid = "generated"
	}
	write(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg, "retryable": retry}, "correlationId": cid})
}
