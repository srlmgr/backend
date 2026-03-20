package conversion

import (
	"errors"
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"

	"github.com/srlmgr/backend/db/dberrors"
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/repository/repoerrors"
)

const (
	importFormatJSON = "json"
	importFormatCSV  = "csv"
)

// Service converts database models to gRPC messages.
type Service struct{}

// New creates a new conversion service.
func New() *Service {
	return &Service{}
}

// ImportFormatsToProto converts persisted import format strings to protobuf enums.
func ImportFormatsToProto(formats []string) []commonv1.ImportFormat {
	if len(formats) == 0 {
		return nil
	}

	out := make([]commonv1.ImportFormat, 0, len(formats))
	for _, format := range formats {
		switch format {
		case importFormatJSON:
			out = append(out, commonv1.ImportFormat_IMPORT_FORMAT_JSON)
		case importFormatCSV:
			out = append(out, commonv1.ImportFormat_IMPORT_FORMAT_CSV)
		default:
			out = append(out, commonv1.ImportFormat_IMPORT_FORMAT_UNSPECIFIED)
		}
	}

	return out
}

// ImportFormatsFromProto converts protobuf enum values to persisted
// import format strings.
func ImportFormatsFromProto(formats []commonv1.ImportFormat) ([]string, error) {
	if len(formats) == 0 {
		return nil, nil
	}

	out := make([]string, 0, len(formats))
	for _, format := range formats {
		switch format {
		case commonv1.ImportFormat_IMPORT_FORMAT_JSON:
			out = append(out, importFormatJSON)
		case commonv1.ImportFormat_IMPORT_FORMAT_CSV:
			out = append(out, importFormatCSV)
		case commonv1.ImportFormat_IMPORT_FORMAT_UNSPECIFIED:
			// Skip unspecified formats.
		default:
			return nil, fmt.Errorf("unsupported import format: %s", format.String())
		}
	}

	return out, nil
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
		SupportedFormats: ImportFormatsToProto(model.SupportedImportFormats),
	}
}

// SeriesToSeries converts a Series model to a Series message.
func (s *Service) SeriesToSeries(model *models.Series) *commonv1.Series {
	if model == nil {
		return nil
	}

	return &commonv1.Series{
		Id:           uint32(model.ID),
		SimulationId: uint32(model.SimulationID),
		Name:         model.Name,
		Description:  model.Description.GetOr(""),
		IsActive:     model.IsActive,
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

func (s *Service) MapErrorToRPCCode(err error) connect.Code {
	// Map specific error types to gRPC codes here.
	if errors.Is(err, repoerrors.ErrNotFound) {
		return connect.CodeNotFound
	}
	if errors.Is(dberrors.RacingSimErrors.ErrUniqueRacingSimsNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.SeriesErrors.ErrUniqueSeriesSimulationIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}

	// If we haven't mapped the error to a specific gRPC code,
	// return Internal for all errors.
	return connect.CodeInternal
}
