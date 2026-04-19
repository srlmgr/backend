package standings

import (
	"cmp"
	"maps"
	"slices"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"

	"github.com/srlmgr/backend/db/models"
)

type (
	Participation interface {
		RaceGridID() int32
		ReferenceID() int32
		FinishPosition() int32
	}
	resultEntryParticipation struct {
		raceGridID     int32
		referenceID    int32
		finishPosition int32
	}
	referenceStandingWork struct {
		referenceID          int32
		standingData         *commonv1.StandingData
		rawPointsByEventID   map[int32]int32
		finishPositionCounts map[int32]int32
		skipEventIDs         []int32
	}
	referenceEventData struct {
		rawPoints      int32
		bonusPoints    int32
		penaltyPoints  int32
		participations []Participation
	}
	ComputedStanding struct {
		ReferenceID  int32
		StandingData *commonv1.StandingData
		SkipEventIDs []int32
	}
	SkipModeType          int
	ComputeStandingsInput struct {
		EventIDs       []int32
		Bookings       []*models.BookingEntry
		Participations []Participation
		NumTotalEvents int
		NumSkip        int
		SkipMode       SkipModeType
		ReferenceID    func(booking *models.BookingEntry) int32
	}
	ComputeStandings struct{}
)

const (
	SkipModeAlways SkipModeType = iota
	SkipModeNever
	SkipModeWhenApplicable
)

func NewComputeStandings() *ComputeStandings {
	return &ComputeStandings{}
}

func (p resultEntryParticipation) RaceGridID() int32     { return p.raceGridID }
func (p resultEntryParticipation) ReferenceID() int32    { return p.referenceID }
func (p resultEntryParticipation) FinishPosition() int32 { return p.finishPosition }

//nolint:whitespace // editor/linter issue
func ParticipationsFromResultEntries(
	resultEntries []*models.ResultEntry,
	referenceID func(entry *models.ResultEntry) int32,
) []Participation {
	if referenceID == nil {
		return nil
	}

	participations := make([]Participation, 0, len(resultEntries))
	for _, entry := range resultEntries {
		if entry == nil {
			continue
		}

		participations = append(participations, resultEntryParticipation{
			raceGridID:     entry.RaceGridID,
			referenceID:    referenceID(entry),
			finishPosition: entry.FinishPosition,
		})
	}

	return participations
}

//nolint:whitespace,gocyclo,funlen // editor/linter issue, lot of logic
func (c *ComputeStandings) Compute(input *ComputeStandingsInput) (
	ret []*ComputedStanding,
) {
	if input == nil || len(input.EventIDs) == 0 || input.ReferenceID == nil {
		return nil
	}

	bookingsByEventID := make(map[int32][]*models.BookingEntry)
	for _, booking := range input.Bookings {
		if booking == nil {
			continue
		}
		bookingsByEventID[booking.EventID] = append(
			bookingsByEventID[booking.EventID],
			booking)
	}

	workByReferenceID := make(map[int32]*referenceStandingWork)
	processedEventIDs := make([]int32, 0, len(input.EventIDs))
	for _, eventID := range input.EventIDs {
		processedEventIDs = append(processedEventIDs, eventID)
		eventBookings := bookingsByEventID[eventID]
		raceGridIDs := make(map[int32]struct{})
		eventDataByReferenceID := make(map[int32]*referenceEventData)

		for _, booking := range eventBookings {
			if booking == nil {
				continue
			}

			if booking.RaceGridID.IsValue() {
				raceGridIDs[booking.RaceGridID.MustGet()] = struct{}{}
			}

			referenceID := input.ReferenceID(booking)
			eventData := getOrCreateReferenceEventData(eventDataByReferenceID, referenceID)
			eventData.rawPoints += booking.Points

			switch string(booking.SourceType) {
			case "qualification_pos", "least_incidents", "fastest_lap", "top_n_finishers":
				eventData.bonusPoints += booking.Points
			case "penalty_points":
				eventData.penaltyPoints += booking.Points
			}
		}

		for _, participation := range input.Participations {
			if participation == nil {
				continue
			}
			if _, ok := raceGridIDs[participation.RaceGridID()]; !ok {
				continue
			}
			referenceID := participation.ReferenceID()
			eventData := getOrCreateReferenceEventData(eventDataByReferenceID, referenceID)
			eventData.participations = append(eventData.participations, participation)
		}

		referenceIDs := make(map[int32]struct{})
		maps.Copy(referenceIDs, keysToSet(workByReferenceID))
		maps.Copy(referenceIDs, keysToSet(eventDataByReferenceID))

		for referenceID := range referenceIDs {
			work := getOrCreateReferenceStandingWork(workByReferenceID, referenceID)
			work.standingData.PrevPosition = work.standingData.Position

			eventData := getOrCreateReferenceEventData(eventDataByReferenceID, referenceID)
			work.rawPointsByEventID[eventID] = eventData.rawPoints

			work.standingData.TotalPoints, work.skipEventIDs = calculateTotalPoints(
				processedEventIDs,
				work.rawPointsByEventID,
				input,
			)
			work.standingData.BonusPoints += eventData.bonusPoints
			work.standingData.PenaltyPoints += eventData.penaltyPoints

			if len(eventData.participations) > 0 {
				work.standingData.NumEvents++
			}
			for _, participation := range eventData.participations {
				work.standingData.NumRaces++
				if participation.FinishPosition() > 0 {
					work.finishPositionCounts[participation.FinishPosition()]++
				}
				if participation.FinishPosition() == 1 {
					work.standingData.NumWins++
				}
				if participation.FinishPosition() <= 3 {
					work.standingData.NumPodiums++
				}
				if participation.FinishPosition() <= 5 {
					work.standingData.NumTop5++
				}
				if participation.FinishPosition() <= 10 {
					work.standingData.NumTop10++
				}
			}
		}

		applyPositions(workByReferenceID)
	}

	return computedStandingsSlice(workByReferenceID)
}

//nolint:whitespace // editor/linter issue
func getOrCreateReferenceStandingWork(
	workByReferenceID map[int32]*referenceStandingWork,
	referenceID int32,
) *referenceStandingWork {
	if work, ok := workByReferenceID[referenceID]; ok {
		return work
	}

	work := &referenceStandingWork{
		referenceID:          referenceID,
		standingData:         &commonv1.StandingData{},
		rawPointsByEventID:   make(map[int32]int32),
		finishPositionCounts: make(map[int32]int32),
	}
	workByReferenceID[referenceID] = work

	return work
}

//nolint:whitespace // editor/linter issue
func getOrCreateReferenceEventData(
	eventDataByReferenceID map[int32]*referenceEventData,
	referenceID int32,
) *referenceEventData {
	if eventData, ok := eventDataByReferenceID[referenceID]; ok {
		return eventData
	}

	eventData := &referenceEventData{}
	eventDataByReferenceID[referenceID] = eventData

	return eventData
}

//nolint:whitespace,funlen // editor/linter issue
func calculateTotalPoints(
	processedEventIDs []int32,
	rawPointsByEventID map[int32]int32,
	input *ComputeStandingsInput,
) (int32, []int32) {
	rawTotal := int32(0)
	for _, eventID := range processedEventIDs {
		rawTotal += rawPointsByEventID[eventID]
	}

	if !shouldApplySkipMode(input) || input.NumSkip <= 0 {
		return rawTotal, nil
	}

	type eventPoints struct {
		eventID int32
		points  int32
		index   int
	}

	events := make([]eventPoints, 0, len(processedEventIDs))
	for index, eventID := range processedEventIDs {
		events = append(events, eventPoints{
			eventID: eventID,
			points:  rawPointsByEventID[eventID],
			index:   index,
		})
	}

	sorted := slices.Clone(events)
	slices.SortFunc(sorted, func(a, b eventPoints) int {
		if diff := cmp.Compare(a.points, b.points); diff != 0 {
			return diff
		}

		return cmp.Compare(a.index, b.index)
	})

	numSkipped := min(input.NumSkip, len(sorted))
	skippedByEventID := make(map[int32]struct{}, numSkipped)
	for _, item := range sorted[:numSkipped] {
		skippedByEventID[item.eventID] = struct{}{}
		rawTotal -= item.points
	}

	skippedEventIDs := make([]int32, 0, numSkipped)
	for _, item := range events {
		if _, ok := skippedByEventID[item.eventID]; ok {
			skippedEventIDs = append(skippedEventIDs, item.eventID)
		}
	}

	return rawTotal, skippedEventIDs
}

func shouldApplySkipMode(input *ComputeStandingsInput) bool {
	if input == nil {
		return false
	}

	//nolint:exhaustive // skip mode is intentionally not exhaustive
	switch input.SkipMode {
	case SkipModeAlways:
		return true
	case SkipModeWhenApplicable:
		return len(input.EventIDs) >= (input.NumTotalEvents - input.NumSkip)
	default:
		return false
	}
}

func applyPositions(workByReferenceID map[int32]*referenceStandingWork) {
	orderedReferenceIDs := slices.Collect(maps.Keys(workByReferenceID))
	slices.SortFunc(orderedReferenceIDs, func(a, b int32) int {
		left := workByReferenceID[a].standingData
		right := workByReferenceID[b].standingData

		if diff := cmp.Compare(right.TotalPoints, left.TotalPoints); diff != 0 {
			return diff
		}
		if diff := compareFinishPositions(
			workByReferenceID[a].finishPositionCounts,
			workByReferenceID[b].finishPositionCounts,
		); diff != 0 {
			return diff
		}
		if diff := cmp.Compare(left.PenaltyPoints, right.PenaltyPoints); diff != 0 {
			return diff
		}
		if diff := cmp.Compare(right.BonusPoints, left.BonusPoints); diff != 0 {
			return diff
		}

		return cmp.Compare(a, b)
	})

	for index, referenceID := range orderedReferenceIDs {
		workByReferenceID[referenceID].standingData.Position = int32(index + 1)
	}
}

func compareFinishPositions(left, right map[int32]int32) int {
	positions := make(map[int32]struct{})
	for position := range left {
		positions[position] = struct{}{}
	}
	for position := range right {
		positions[position] = struct{}{}
	}

	orderedPositions := slices.Collect(maps.Keys(positions))
	slices.Sort(orderedPositions)
	for _, position := range orderedPositions {
		if diff := cmp.Compare(right[position], left[position]); diff != 0 {
			return diff
		}
	}

	return 0
}

//nolint:whitespace // editor/linter issue
func computedStandingsSlice(
	workByReferenceID map[int32]*referenceStandingWork,
) []*ComputedStanding {
	orderedReferenceIDs := slices.Collect(maps.Keys(workByReferenceID))
	slices.SortFunc(orderedReferenceIDs, func(a, b int32) int {
		left := workByReferenceID[a].standingData
		right := workByReferenceID[b].standingData

		if diff := cmp.Compare(left.Position, right.Position); diff != 0 {
			return diff
		}

		return cmp.Compare(a, b)
	})

	result := make([]*ComputedStanding, 0, len(orderedReferenceIDs))
	for _, referenceID := range orderedReferenceIDs {
		work := workByReferenceID[referenceID]
		result = append(result, &ComputedStanding{
			ReferenceID:  work.referenceID,
			StandingData: work.standingData,
			SkipEventIDs: work.skipEventIDs,
		})
	}

	return result
}

func keysToSet[V any](items map[int32]V) map[int32]struct{} {
	set := make(map[int32]struct{}, len(items))
	for key := range items {
		set[key] = struct{}{}
	}

	return set
}
