package presentator

import "context"

type Service struct {
	store     *Store
	predictor PredictorXPort
	pdf       PDFPort
	queue     chan work
}
type work struct {
	jobID      string
	generation *GenerationRequest
}

func NewService(store *Store, p PredictorXPort, pdf PDFPort) *Service {
	return &Service{store: store, predictor: p, pdf: pdf, queue: make(chan work, 32)}
}
func (s *Service) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case w := <-s.queue:
			s.run(ctx, w)
		}
	}
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
		s.queue <- work{jobID: j.ID, generation: &req}
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
		s.queue <- work{jobID: j.ID}
	}
	return j, nil
}
func (s *Service) run(ctx context.Context, w work) {
	j, err := s.store.Job(w.jobID)
	if err != nil {
		return
	}
	j.Status = "running"
	s.store.UpdateJob(j)
	p, err := s.store.Project(j.ProjectID)
	if err == nil && w.generation != nil {
		var d Deck
		d, err = s.predictor.Generate(ctx, *w.generation, p.Deck)
		if err == nil {
			err = d.Validate()
		}
		if err == nil {
			j.Candidate = &d
		}
	} else if err == nil {
		j.Artifact, err = s.pdf.Render(ctx, p.Deck)
	}
	if err != nil {
		j.Status = "failed"
		j.Error = &JobError{Code: "internal", Message: "job failed", Retryable: true}
	} else {
		j.Status = "succeeded"
	}
	s.store.UpdateJob(j)
}
func (s *Service) Apply(jobID string, base int) (Project, error) {
	j, err := s.store.Job(jobID)
	if err != nil {
		return Project{}, err
	}
	if j.Status != "succeeded" || j.Candidate == nil {
		return Project{}, ErrConflict
	}
	return s.store.SaveDeck(j.ProjectID, base, *j.Candidate)
}
