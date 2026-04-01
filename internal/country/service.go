package country

import (
	"context"
	"errors"
)

type Service interface {
	List(ctx context.Context, limit, offset int) ([]Country, int64, error)
	Get(ctx context.Context, id int64) (Country, error)
	Create(ctx context.Context, req Country) (Country, error)
	Update(ctx context.Context, id int64, patch UpdatePatch) (Country, error)
	Delete(ctx context.Context, id int64) error
}

type DefaultService struct {
	repo Repository
}

func NewService(repo Repository) *DefaultService {
	return &DefaultService{repo: repo}
}

func (s *DefaultService) List(ctx context.Context, limit, offset int) ([]Country, int64, error) {
	if s.repo == nil {
		return nil, 0, errors.New("database connection is not configured")
	}

	return s.repo.List(ctx, limit, offset)
}

func (s *DefaultService) Get(ctx context.Context, id int64) (Country, error) {
	if s.repo == nil {
		return Country{}, errors.New("database connection is not configured")
	}

	return s.repo.Get(ctx, id)
}

func (s *DefaultService) Create(ctx context.Context, req Country) (Country, error) {
	if s.repo == nil {
		return Country{}, errors.New("database connection is not configured")
	}

	return s.repo.Create(ctx, req)
}

func (s *DefaultService) Update(ctx context.Context, id int64, patch UpdatePatch) (Country, error) {
	if s.repo == nil {
		return Country{}, errors.New("database connection is not configured")
	}

	return s.repo.Update(ctx, id, patch)
}

func (s *DefaultService) Delete(ctx context.Context, id int64) error {
	if s.repo == nil {
		return errors.New("database connection is not configured")
	}

	return s.repo.Delete(ctx, id)
}
