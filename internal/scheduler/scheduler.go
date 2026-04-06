package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
)

type Task func(context.Context)

type Job struct {
	Name     string
	Interval time.Duration
	Task     Task
}

type Scheduler struct {
	logger *log.Logger
	mu     sync.RWMutex
	jobs   []Job
}

func New(logger *log.Logger) *Scheduler {
	return &Scheduler{logger: logger}
}

func (s *Scheduler) Add(name string, interval time.Duration, task Task) {
	if task == nil || interval <= 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs = append(s.jobs, Job{
		Name:     name,
		Interval: interval,
		Task:     task,
	})
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.RLock()
	jobs := append([]Job(nil), s.jobs...)
	s.mu.RUnlock()

	for _, job := range jobs {
		job := job
		go s.runJob(ctx, job)
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	s.runTask(ctx, job)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runTask(ctx, job)
		}
	}
}

func (s *Scheduler) runTask(ctx context.Context, job Job) {
	if s.logger != nil {
		s.logger.Printf("scheduler job started: %s interval=%s", job.Name, job.Interval)
	}
	job.Task(ctx)
	if s.logger != nil {
		s.logger.Printf("scheduler job finished: %s", job.Name)
	}
}
