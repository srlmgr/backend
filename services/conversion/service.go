package conversion

import (
	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"

	"github.com/srlmgr/backend/db/models"
)

// Service converts database models to gRPC messages.
type Service struct{}

// New creates a new conversion service.
func New() *Service {
	return &Service{}
}

// RacingSimToSimulation converts a RacingSim model to a Simulation message.
func (s *Service) RacingSimToSimulation(model *models.RacingSim) *commonv1.Simulation {
	if model == nil {
		return nil
	}

	return &commonv1.Simulation{
		Id:               uint32(model.ID),
		Name:             model.Name,
		IsActive:         model.IsActive,
		SupportedFormats: append([]string(nil), model.SupportedImportFormats...),
	}
}

// RacingSimsToSimulations converts RacingSim models to Simulation messages.
//
//nolint:lll // readability
func (s *Service) RacingSimsToSimulations(items []*models.RacingSim) []*commonv1.Simulation {
	if len(items) == 0 {
		return []*commonv1.Simulation{}
	}

	out := make([]*commonv1.Simulation, 0, len(items))
	for _, item := range items {
		converted := s.RacingSimToSimulation(item)
		if converted != nil {
			out = append(out, converted)
		}
	}

	return out
}
