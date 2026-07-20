package presentator

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

type Service struct {
	store      *Store
	predictor  PredictorXPort
	pdf        PDFPort
	queue      chan work
	workers    int
	jobTimeout time.Duration
}
type work struct {
	jobID      string
	generation *GenerationRequest
}

func NewService(store *Store, p PredictorXPort, pdf PDFPort) *Service {
	return &Service{store: store, predictor: p, pdf: pdf, queue: make(chan work, 32), workers: 4, jobTimeout: 30 * time.Second}
}
func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range s.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case w := <-s.queue:
					s.run(ctx, w)
				}
			}
		}()
	}
	<-ctx.Done()
	wg.Wait()
}
func (s *Service) EnqueueGeneration(projectID, key string, req GenerationRequest) (Job, error) {
	p, err := s.store.Project(projectID)
	if err != nil {
		return Job{}, err
	}
	j, reused, err := s.store.CreateJob(projectID, "generation", key, p.Revision)
	if err != nil {
		return Job{}, err
	}
	if !reused {
		select {
		case s.queue <- work{jobID: j.ID, generation: &req}:
		default:
			j.Status = "failed"
			j.Error = &JobError{Code: "queue_full", Message: "job queue is full", Retryable: true}
			s.store.UpdateJob(j)
			return Job{}, ErrQueueFull
		}
	}
	return j, nil
}
func (s *Service) EnqueueExport(projectID, key string, revision int) (Job, error) {
	p, err := s.store.Project(projectID)
	if err != nil {
		return Job{}, err
	}
	if p.Revision != revision {
		return Job{}, ErrConflict
	}
	j, reused, err := s.store.CreateJob(projectID, "export", key, revision)
	if err != nil {
		return Job{}, err
	}
	if !reused {
		select {
		case s.queue <- work{jobID: j.ID}:
		default:
			j.Status = "failed"
			j.Error = &JobError{Code: "queue_full", Message: "job queue is full", Retryable: true}
			s.store.UpdateJob(j)
			return Job{}, ErrQueueFull
		}
	}
	return j, nil
}
func (s *Service) run(ctx context.Context, w work) {
	jobCtx, cancel := context.WithTimeout(ctx, s.jobTimeout)
	defer cancel()
	j, err := s.store.Job(w.jobID)
	if err != nil {
		return
	}
	j.Status = "running"
	s.store.UpdateJob(j)
	p, err := s.store.Project(j.ProjectID)
	if err == nil && w.generation != nil {
		var d Deck
		d, err = s.predictor.Generate(jobCtx, *w.generation, p.Deck)
		if err == nil {
			d.Normalize()
			err = d.Validate()
		}
		if err == nil {
			j.Candidate = &d
		}
	} else if err == nil {
		j.Artifact, err = s.pdf.Render(jobCtx, p.Deck)
	}
	if err != nil {
		j.Status = "failed"
		code := "internal"
		if jobCtx.Err() != nil {
			code = "timeout"
		}
		j.Error = &JobError{Code: code, Message: "job failed", Retryable: true}
	} else {
		j.Status = "succeeded"
	}
	s.store.UpdateJob(j)
}
func (s *Service) Apply(jobID, key string, base int) (Project, error) {
	j, err := s.store.Job(jobID)
	if err != nil {
		return Project{}, err
	}
	if j.Status != "succeeded" || j.Candidate == nil {
		return Project{}, ErrConflict
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%d", jobID, base))))
	return s.store.ApplyCandidate(jobID, key, fingerprint, base, *j.Candidate)
}
