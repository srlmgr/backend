package impl

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/service"
)

type (
	participantsImpl struct {
		r      repository.Repository
		logger *log.Logger
		tracer trace.Tracer
	}
)

var _ service.ParticipantsService = (*participantsImpl)(nil)

//nolint:whitespace //editor/linter issue
func newParticipantsImpl(
	r repository.Repository,
	logger *log.Logger,
	tracer trace.Tracer,
) *participantsImpl {
	return &participantsImpl{
		r:      r,
		logger: logger,
		tracer: tracer,
	}
}

//nolint:whitespace //editor/linter issue
func (s *participantsImpl) GetDrivers(
	ctx context.Context,
	seasonID int,
) ([]*models.SeasonDriver, error) {
	return s.r.Drivers().SeasonDrivers().LoadBySeasonID(ctx, int32(seasonID))
}

//nolint:whitespace //editor/linter issue
func (s *participantsImpl) GetTeams(
	ctx context.Context,
	seasonID int,
) ([]*models.Team, error) {
	return s.r.Teams().Teams().LoadBySeasonID(ctx, int32(seasonID))
}
