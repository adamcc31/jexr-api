package usecase

import "context"

type HealthUsecase interface {
	Check(ctx context.Context) map[string]string
}

type healthUsecase struct{}

func NewHealthUsecase() HealthUsecase {
	return &healthUsecase{}
}

func (u *healthUsecase) Check(ctx context.Context) map[string]string {
	return map[string]string{
		"status": "ok",
	}
}
