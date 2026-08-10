package impl

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
)

type (
	serviceImpl struct {
		r      repository.Repository
		logger *log.Logger
		tracer trace.Tracer
	}
)

var _ service.Service = (*serviceImpl)(nil)

// creates a new service implementation based on the provided
// repository, logger, and tracer.
// this implementation is used on a running production instance
//
//nolint:whitespace //editor/linter issue
func New(
	r repository.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
) service.Service {
	return &serviceImpl{
		r:      r,
		logger: logger,
		tracer: tracer,
	}
}

func (s *serviceImpl) StandingsService() service.StandingsService {
	return newStandingsImpl(s.r, s.logger, s.tracer)
}

func (s *serviceImpl) ParticipantsService() service.ParticipantsService {
	return newParticipantsImpl(s.r, s.logger, s.tracer)
}

func (s *serviceImpl) SummaryService() service.SummaryService {
	return newSummaryImpl(s.r, s.logger, s.tracer)
}

func (s *serviceImpl) BookingsService() service.BookingsService {
	return nil
}
