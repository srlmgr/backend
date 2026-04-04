package points

import (
	"fmt"
)

type (
	PointType      float64
	SeasonSettings struct {
		Pointsystem PointSystemSettings
		Standings   StandingsSettings
	}
	StandingsSettings struct {
		SkipResults int32
		PerClass    bool
	}
	PointSystemSettings struct {
		Eligibility EligibilitySettings
		Policies    []RankedPolicySettings
		Races       []RaceSettings
	}
	EligibilitySettings struct {
		RaceDistPct float64
		Guests      bool
	}

	RankedPolicySettings struct {
		Name   string
		Points map[PointPolicyType][]PointType
	}
	PointPenaltySettings struct {
		Name      string
		Arguments map[PointPolicyType]any
	}
	ThresholdPenaltySettings struct {
		Threshold  int32
		PenaltyPct float64
	}
	RaceSettings struct {
		Name            string
		Policies        []PointPolicyType
		AwardSettings   []RankedPolicySettings
		PenaltySettings []PointPenaltySettings
	}

	PointPolicyType int
	PointPolicy     interface {
		Type() PointPolicyType
	}
	Input interface {
		FinishPosition() int32
		QualiPosition() int32
		ClassID() int32
		DriverID() int32
		TeamID() int32
		IsGuest() bool
		Incidents() int32
		LapsCompleted() int32
		FastestLap() int32
		ReferenceID() int32
	}
	InputOpt         func(*defaultInputImpl) *defaultInputImpl
	defaultInputImpl struct {
		finishPosition int32
		qualiPosition  int32
		classID        int32
		driverID       int32
		teamID         int32
		isGuest        bool
		incidents      int32
		lapsCompleted  int32
		fastestLap     int32
		referenceID    int32
	}
)

const (
	PointsPolicyFinishPos PointPolicyType = iota
	PointsPolicyFastestLap
	PointsPolicyLeastIncidents
	PointsPolicyIncidentsExceeded
	PointsPolicyQualificationPos
	PointsPolicyTopNFinishers
	PointsPolicyCustom
)

var (
	_                       Input = (*defaultInputImpl)(nil)
	stringToPointPolicyType       = map[string]PointPolicyType{
		"finish_pos":         PointsPolicyFinishPos,
		"fastest_lap":        PointsPolicyFastestLap,
		"least_incidents":    PointsPolicyLeastIncidents,
		"incidents_exceeded": PointsPolicyIncidentsExceeded,
		"qualification_pos":  PointsPolicyQualificationPos,
		"top_n_finishers":    PointsPolicyTopNFinishers,
		"custom":             PointsPolicyCustom,
	}
	pointPolicyTypeToString = func() map[PointPolicyType]string {
		m := make(map[PointPolicyType]string)
		for str, val := range stringToPointPolicyType {
			m[val] = str
		}
		return m
	}()
)

func (p PointPolicyType) MarshalText() ([]byte, error) {
	str, ok := pointPolicyTypeToString[p]
	if !ok {
		return nil, fmt.Errorf("invalid PointPolicyType: %d", p)
	}
	return []byte(str), nil
}

func (p *PointPolicyType) UnmarshalText(text []byte) error {
	str := string(text)
	val, ok := stringToPointPolicyType[str]
	if !ok {
		return fmt.Errorf("invalid PointPolicyType: %s", str)
	}
	*p = val
	return nil
}

func (p PointPolicyType) String() string {
	str, ok := pointPolicyTypeToString[p]
	if !ok {
		return fmt.Sprintf("unknown(%d)", p)
	}
	return str
}

func NewInput(opts ...InputOpt) Input {
	input := &defaultInputImpl{}
	for _, opt := range opts {
		opt(input)
	}
	return input
}

func WithFinishPosition(pos int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.finishPosition = pos
		return i
	}
}

func WithQualiPosition(pos int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.qualiPosition = pos
		return i
	}
}

func WithClassID(id int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.classID = id
		return i
	}
}

func WithDriverID(id int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.driverID = id
		return i
	}
}

func WithTeamID(id int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.teamID = id
		return i
	}
}

func WithIsGuest(isGuest bool) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.isGuest = isGuest
		return i
	}
}

func WithIncidents(incidents int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.incidents = incidents
		return i
	}
}

func WithLapsCompleted(laps int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.lapsCompleted = laps
		return i
	}
}

func WithFastestLap(lap int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.fastestLap = lap
		return i
	}
}

func WithReferenceID(id int32) InputOpt {
	return func(i *defaultInputImpl) *defaultInputImpl {
		i.referenceID = id
		return i
	}
}

func (i defaultInputImpl) FinishPosition() int32 { return i.finishPosition }
func (i defaultInputImpl) QualiPosition() int32  { return i.qualiPosition }
func (i defaultInputImpl) ReferenceID() int32    { return i.referenceID }
func (i defaultInputImpl) ClassID() int32        { return i.classID }
func (i defaultInputImpl) DriverID() int32       { return i.driverID }
func (i defaultInputImpl) TeamID() int32         { return i.teamID }
func (i defaultInputImpl) IsGuest() bool         { return i.isGuest }
func (i defaultInputImpl) Incidents() int32      { return i.incidents }
func (i defaultInputImpl) LapsCompleted() int32  { return i.lapsCompleted }
func (i defaultInputImpl) FastestLap() int32     { return i.fastestLap }
