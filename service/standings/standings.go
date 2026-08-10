package standings

type (
	StandingData struct {
		Position       int32
		PrevPosition   int32
		TotalPoints    int32
		BonusPoints    int32
		PenaltyPoints  int32
		NumEvents      int32
		NumRaces       int32
		NumPenaltyFree int32
		NumWins        int32
		NumPodiums     int32
		NumTop5        int32
		NumTop10       int32
	}
)
