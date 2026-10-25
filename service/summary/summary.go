package summary

import "github.com/srlmgr/backend/db/models"

type (
	BaseSummary struct {
		Season    *models.Season
		Primary   []*SummaryEntry
		Secondary []*SummaryEntry
	}
	RaceSummary struct {
		RaceID  int
		Summary *BaseSummary
	}
	EventSummary struct {
		EventID int
		Summary *BaseSummary
	}
	PointSummary struct {
		TotalPoints   int // TotalPoints = RawPoints + BonusPoints - PenaltyPoints
		RawPoints     int
		BonusPoints   int
		PenaltyPoints int
	}

	SummaryEntry struct {
		ReferenceID int
		CarClassID  int
		Points      PointSummary
	}
)

func (s *SummaryEntry) AddFrom(other *PointSummary) {
	s.Points.TotalPoints += other.TotalPoints
	s.Points.RawPoints += other.RawPoints
	s.Points.BonusPoints += other.BonusPoints
	s.Points.PenaltyPoints += other.PenaltyPoints
}
