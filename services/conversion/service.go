//nolint:dupl // due to nature of conversion code
package conversion

import (
	"errors"
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/db/dberrors"
	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/repository/repoerrors"
)

// Service converts database models to gRPC messages.
type Service struct {
	logger *log.Logger
}

// New creates a new conversion service.
func New() *Service {
	return &Service{logger: log.New()}
}

// ImportFormatsToProto converts persisted import format strings to protobuf enums.
func ImportFormatsToProto(formats []string) []commonv1.ImportFormat {
	if len(formats) == 0 {
		return nil
	}

	out := make([]commonv1.ImportFormat, 0, len(formats))
	for _, format := range formats {
		switch format {
		case ImportFormatJSON:
			out = append(out, commonv1.ImportFormat_IMPORT_FORMAT_JSON)
		case ImportFormatCSV:
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
			out = append(out, ImportFormatJSON)
		case commonv1.ImportFormat_IMPORT_FORMAT_CSV:
			out = append(out, ImportFormatCSV)
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

// PointSystemToPointSystem converts a PointSystem model to a PointSystem message.
//
//nolint:lll // readability
func (s *Service) PointSystemToPointSystem(model *models.PointSystem) *commonv1.PointSystem {
	if model == nil {
		return nil
	}

	return &commonv1.PointSystem{
		Id:          uint32(model.ID),
		Name:        model.Name,
		Description: model.Description.GetOr(""),
	}
}

// PointRuleToPointRule converts a PointRule model to a PointRule message.
// The full conversion from MetadataJSON to proto fields is deferred to
// a follow-up issue.
func (s *Service) PointRuleToPointRule(_ *models.PointRule) *commonv1.PointRule {
	return &commonv1.PointRule{}
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

// SeasonToSeason converts a Season model to a Season message.
func (s *Service) SeasonToSeason(model *models.Season) *commonv1.Season {
	if model == nil {
		return nil
	}

	return &commonv1.Season{
		Id:            uint32(model.ID),
		SeriesId:      uint32(model.SeriesID),
		Name:          model.Name,
		PointSystemId: uint32(model.PointSystemID),
		HasTeams:      model.HasTeams,
		SkipEvents:    model.SkipEvents,
		Status:        model.Status,
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

// TrackToTrack converts a Track model to a Track message.
func (s *Service) TrackToTrack(model *models.Track) *commonv1.Track {
	if model == nil {
		return nil
	}

	return &commonv1.Track{
		Id:         uint32(model.ID),
		Name:       model.Name,
		Country:    model.Country.GetOr(""),
		Latitude:   model.Latitude.GetOr(decimal.Zero).InexactFloat64(),
		Longitude:  model.Longitude.GetOr(decimal.Zero).InexactFloat64(),
		WebsiteUrl: model.WebsiteURL.GetOr(""),
	}
}

// TrackLayoutToTrackLayout converts a TrackLayout model to a TrackLayout message.
//
//nolint:lll // readability
func (s *Service) TrackLayoutToTrackLayout(model *models.TrackLayout) *commonv1.TrackLayout {
	if model == nil {
		return nil
	}

	return &commonv1.TrackLayout{
		Id:             uint32(model.ID),
		TrackId:        uint32(model.TrackID),
		Name:           model.Name,
		LengthMeters:   model.LengthMeters.GetOr(0),
		LayoutImageUrl: model.LayoutImageURL.GetOr(""),
	}
}

// CarManufacturerToCarManufacturer converts a CarManufacturer model to a
// CarManufacturer message.
//
//nolint:whitespace // editor/linter issue
func (s *Service) CarManufacturerToCarManufacturer(
	model *models.CarManufacturer,
) *commonv1.CarManufacturer {
	if model == nil {
		return nil
	}

	return &commonv1.CarManufacturer{
		Id:   uint32(model.ID),
		Name: model.Name,
	}
}

// CarBrandToCarBrand converts a CarBrand model to a CarBrand message.
func (s *Service) CarBrandToCarBrand(model *models.CarBrand) *commonv1.CarBrand {
	if model == nil {
		return nil
	}

	return &commonv1.CarBrand{
		Id:             uint32(model.ID),
		ManufacturerId: uint32(model.ManufacturerID),
		Name:           model.Name,
	}
}

// CarModelToCarModel converts a CarModel model to a CarModel message.
func (s *Service) CarModelToCarModel(model *models.CarModel) *commonv1.CarModel {
	if model == nil {
		return nil
	}

	return &commonv1.CarModel{
		Id:      uint32(model.ID),
		BrandId: uint32(model.BrandID),
		Name:    model.Name,
	}
}

// DriverToDriver converts a Driver model to a Driver message.
func (s *Service) DriverToDriver(model *models.Driver) *commonv1.Driver {
	if model == nil {
		return nil
	}

	return &commonv1.Driver{
		Id:         uint32(model.ID),
		ExternalId: model.ExternalID,
		Name:       model.Name,
		IsActive:   model.IsActive,
	}
}

// TeamToTeam converts a Team model to a Team message.
func (s *Service) TeamToTeam(model *models.Team) *commonv1.Team {
	if model == nil {
		return nil
	}

	return &commonv1.Team{
		Id:       uint32(model.ID),
		SeasonId: uint32(model.SeasonID),
		Name:     model.Name,
		IsActive: model.IsActive,
	}
}

// EventToEvent converts an Event model to an Event message.
//
//nolint:funlen // many fields to convert and validate
func (s *Service) EventToEvent(model *models.Event) *commonv1.Event {
	if model == nil {
		return nil
	}

	event := &commonv1.Event{
		Id:            uint32(model.ID),
		SeasonId:      uint32(model.SeasonID),
		TrackLayoutId: uint32(model.TrackLayoutID),
		Name:          model.Name,
		EventDate:     timestamppb.New(model.EventDate),
	}

	status := model.Status
	if !isKnownEventStatus(status) {
		s.logger.Warn("unknown event status, mapping to UNSPECIFIED",
			log.String("status", model.Status),
			log.Int32("event_id", model.ID),
		)
		status = ""
	}
	if err := SetProtoFieldStringOrEnum(
		event.ProtoReflect(), "status", status); err != nil {
		s.logger.Warn("unknown event status, mapping to UNSPECIFIED",
			log.String("status", status),
			log.Int32("event_id", model.ID),
			log.ErrorField(err),
		)
	}

	processingState := model.ProcessingState
	if !isKnownEventProcessingState(processingState) {
		s.logger.Warn("unknown event processing_state, mapping to UNSPECIFIED",
			log.String("processing_state", model.ProcessingState),
			log.Int32("event_id", model.ID),
		)
		processingState = ""
	}
	if err := SetProtoFieldStringOrEnum(
		event.ProtoReflect(),
		"processing_state",
		processingState,
	); err != nil {
		s.logger.Warn("unknown event processing_state, mapping to UNSPECIFIED",
			log.String("processing_state", processingState),
			log.Int32("event_id", model.ID),
			log.ErrorField(err),
		)
	}

	return event
}

func isKnownEventStatus(status string) bool {
	switch status {
	case EventStatusScheduled, EventStatusCompleted, EventStatusCancelled:
		return true
	default:
		return false
	}
}

func isKnownEventProcessingState(processingState string) bool {
	switch processingState {
	case EventProcessingStateDraft,
		EventProcessingStateRawImported,
		EventProcessingStatePreprocessed,
		EventProcessingStateDriverEntriesComputed,
		EventProcessingStateTeamEntriesComputed,
		EventProcessingStateFinalized:
		return true
	default:
		return false
	}
}

// ResultEntryToResultEntry converts a ResultEntry model to a ResultEntry message.
//
//nolint:lll // readability
func (s *Service) ResultEntryToResultEntry(model *models.ResultEntry) *commonv1.ResultEntry {
	if model == nil {
		return nil
	}

	entry := &commonv1.ResultEntry{
		Id:                uint32(model.ID),
		RaceId:            uint32(model.RaceID),
		DriverId:          uint32(model.DriverID.GetOr(0)),
		TeamId:            uint32(model.TeamID.GetOr(0)),
		CarModelId:        uint32(model.CarModelID.GetOr(0)),
		CarClassId:        uint32(model.CarClassID.GetOr(0)),
		CarNumber:         model.CarNumber.GetOr(""),
		StartingPosition:  model.StartPosition.GetOr(0),
		FinishingPosition: model.FinishPosition,
		CompletedLaps:     model.LapsCompleted,
		QualiTimeMs:       model.QualiLapTimeMS.GetOr(0),
		FastestLapTimeMs:  model.FastestLapTimeMS.GetOr(0),
		TotalTimeMs:       model.TotalTimeMS.GetOr(0),
		Incidents:         model.Incidents.GetOr(0),
		AdminNotes:        model.AdminNotes.GetOr(""),
		RawDriverName:     model.RawDriverName.GetOr(""),
		RawTeamName:       model.RawTeamName.GetOr(""),
		IsGuestDriver:     model.IsGuestDriver,
	}

	switch model.State {
	case ResultStateNormal:
		entry.State = commonv1.ResultEntryState_RESULT_ENTRY_STATE_NORMAL
	case ResultStateDQ:
		entry.State = commonv1.ResultEntryState_RESULT_ENTRY_STATE_DQ
	case ResultStateMappingError:
		entry.State = commonv1.ResultEntryState_RESULT_ENTRY_STATE_MAPPING_ERROR
	default:
		entry.State = commonv1.ResultEntryState_RESULT_ENTRY_STATE_UNSPECIFIED
	}

	return entry
}

// RaceToRace converts a Race model to a Race message.
// Unknown session_type strings map to UNSPECIFIED and emit a warning log.
func (s *Service) RaceToRace(model *models.Race) *commonv1.Race {
	if model == nil {
		return nil
	}

	var sessionType commonv1.RaceSessionType
	switch model.SessionType {
	case RaceSessionTypeQualifying:
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING
	case RaceSessionTypeHeat:
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_HEAT
	case RaceSessionTypeRace:
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE
	default:
		s.logger.Warn("unknown race session_type, mapping to UNSPECIFIED",
			log.String("session_type", model.SessionType),
			log.Int32("race_id", model.ID),
		)
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_UNSPECIFIED
	}

	return &commonv1.Race{
		Id:          uint32(model.ID),
		EventId:     uint32(model.EventID),
		Name:        model.Name,
		SessionType: sessionType,
		SequenceNo:  model.SequenceNo,
	}
}

// RaceGridToRaceGrid converts a RaceGrid model to a RaceGrid message.
// Unknown session_type strings map to UNSPECIFIED and emit a warning log.
func (s *Service) RaceGridToRaceGrid(model *models.RaceGrid) *commonv1.RaceGrid {
	if model == nil {
		return nil
	}

	var sessionType commonv1.RaceSessionType
	switch model.SessionType {
	case RaceSessionTypeQualifying:
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_QUALIFYING
	case RaceSessionTypeHeat:
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_HEAT
	case RaceSessionTypeRace:
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_RACE
	default:
		s.logger.Warn("unknown race grid session_type, mapping to UNSPECIFIED",
			log.String("session_type", model.SessionType),
			log.Int32("race_grid_id", model.ID),
		)
		sessionType = commonv1.RaceSessionType_RACE_SESSION_TYPE_UNSPECIFIED
	}

	return &commonv1.RaceGrid{
		Id:          uint32(model.ID),
		RaceId:      uint32(model.RaceID),
		Name:        model.Name,
		SessionType: sessionType,
		SequenceNo:  model.SequenceNo,
	}
}

//nolint:gocyclo,funlen // many different error types to check for
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
	if errors.Is(dberrors.PointSystemErrors.ErrUniquePointSystemsNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.CarManufacturerErrors.ErrUniqueCarManufacturersNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.CarBrandErrors.ErrUniqueCarBrandsManufacturerIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.CarModelErrors.ErrUniqueCarModelsBrandIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.TrackErrors.ErrUniqueTracksNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.TrackLayoutErrors.ErrUniqueTrackLayoutsTrackIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.SeasonErrors.ErrUniqueSeasonsSeriesIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.EventErrors.ErrUniqueEventsSeasonIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.DriverErrors.ErrUniqueDriversExternalIdUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.TeamErrors.ErrUniqueTeamsSeasonIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.RaceErrors.ErrUniqueRacesEventIdNameUnique, err) {
		return connect.CodeAlreadyExists
	}
	if errors.Is(dberrors.RaceErrors.ErrUniqueRacesEventIdSequenceNoUnique, err) {
		return connect.CodeAlreadyExists
	}
	//nolint:staticcheck // by design
	if errors.Is(dberrors.ErrUniqueResultEntriesRaceIDDriverIDUnique, err) {
		return connect.CodeAlreadyExists
	}

	// If we haven't mapped the error to a specific gRPC code,
	// return Internal for all errors.
	return connect.CodeInternal
}
