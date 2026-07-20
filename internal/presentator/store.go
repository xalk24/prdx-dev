package presentator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

type Store struct {
	mu          sync.RWMutex
	seq         atomic.Uint64
	projects    map[string]Project
	jobs        map[string]Job
	idempotency map[string]string
	applies     map[string]applyRecord
	assets      map[string]Asset
	assetBytes  map[string]int
}

type applyRecord struct {
	fingerprint string
	project     Project
}

func NewStore() *Store {
	return &Store{projects: map[string]Project{}, jobs: map[string]Job{}, idempotency: map[string]string{}, applies: map[string]applyRecord{}, assets: map[string]Asset{}, assetBytes: map[string]int{}}
}
func (s *Store) id(prefix string) string { return fmt.Sprintf("%s-%d", prefix, s.seq.Add(1)) }
func (s *Store) CreateProject(title string, deck Deck) Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := Project{ID: s.id("project"), Title: title, Revision: 1, Deck: deck}
	p.Deck.DeckID = p.ID
	s.projects[p.ID] = p
	return p
}
func (s *Store) Project(id string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	return p, nil
}
func (s *Store) SaveDeck(id string, expected int, deck Deck) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[id]
	if !ok {
		return Project{}, ErrNotFound
	}
	if p.Revision != expected {
		return Project{}, ErrConflict
	}
	p.Revision++
	deck.DeckID = id
	p.Deck = deck
	s.projects[id] = p
	return p, nil
}
func (s *Store) CreateJob(projectID, typ, key string, base int) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[projectID]; !ok {
		return Job{}, false, ErrNotFound
	}
	ik := projectID + ":" + typ + ":" + key
	if id, ok := s.idempotency[ik]; ok {
		return s.jobs[id], true, nil
	}
	for _, j := range s.jobs {
		if j.ProjectID == projectID && j.Type == typ && (j.Status == "queued" || j.Status == "running") {
			return Job{}, false, errorsConflictJob
		}
	}
	j := Job{ID: s.id("job"), ProjectID: projectID, Type: typ, Status: "queued", BaseRevision: base}
	s.jobs[j.ID] = j
	s.idempotency[ik] = j.ID
	return j, false, nil
}

var errorsConflictJob = fmt.Errorf("active job: %w", ErrConflict)

func (s *Store) Job(id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	return j, nil
}
func (s *Store) UpdateJob(j Job) { s.mu.Lock(); defer s.mu.Unlock(); s.jobs[j.ID] = j }

func (s *Store) ApplyCandidate(jobID, key, fingerprint string, base int, deck Deck) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idempotencyID := jobID + ":" + key
	if previous, ok := s.applies[idempotencyID]; ok {
		if previous.fingerprint != fingerprint {
			return Project{}, ErrIdempotencyConflict
		}
		return previous.project, nil
	}
	job, ok := s.jobs[jobID]
	if !ok {
		return Project{}, ErrNotFound
	}
	project, ok := s.projects[job.ProjectID]
	if !ok {
		return Project{}, ErrNotFound
	}
	if project.Revision != base {
		return Project{}, ErrConflict
	}
	project.Revision++
	deck.DeckID = project.ID
	project.Deck = deck
	s.projects[project.ID] = project
	s.applies[idempotencyID] = applyRecord{fingerprint: fingerprint, project: project}
	return project, nil
}

const MaxProjectAssetBytes = 100 << 20

func (s *Store) CreateAsset(projectID, mime string, data []byte) (Asset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[projectID]; !ok {
		return Asset{}, ErrNotFound
	}
	if s.assetBytes[projectID]+len(data) > MaxProjectAssetBytes {
		return Asset{}, ErrConflict
	}
	asset := Asset{ID: s.id("asset"), ProjectID: projectID, MIME: mime, Size: len(data), Data: append([]byte(nil), data...)}
	s.assets[asset.ID] = asset
	s.assetBytes[projectID] += len(data)
	return asset, nil
}

func (s *Store) Asset(projectID, assetID string) (Asset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asset, ok := s.assets[assetID]
	if !ok || asset.ProjectID != projectID {
		return Asset{}, ErrNotFound
	}
	asset.Data = append([]byte(nil), asset.Data...)
	return asset, nil
}

func (s *Store) RendererAssets(projectID string, deck Deck) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := map[string]string{}
	for _, slide := range deck.Slides {
		for _, raw := range slide.Elements {
			var ref struct {
				Type    string `json:"type"`
				AssetID string `json:"assetId"`
			}
			if err := json.Unmarshal(raw, &ref); err != nil {
				return nil, err
			}
			if ref.Type != "image" {
				continue
			}
			asset, ok := s.assets[ref.AssetID]
			if !ok || asset.ProjectID != projectID {
				return nil, ErrAssetMissing
			}
			result[asset.ID] = "data:" + asset.MIME + ";base64," + base64.StdEncoding.EncodeToString(asset.Data)
		}
	}
	return result, nil
}
