//nolint:tagliatelle // external definition
package iracing

import "time"

// Simsession type constants identify the kind of sub-session within an iRacing event.
const (
	// SimsessionTypeLoneQualifying identifies a lone qualifying session.
	SimsessionTypeLoneQualifying = 4
	// SimsessionTypeOpenQualifying identifies an open qualifying session.
	SimsessionTypeOpenQualifying = 5
	// SimsessionTypeRace identifies a race session.
	SimsessionTypeRace = 6
)

// EventResultEnvelope is the top-level wrapper returned by the iRacing API.
type EventResultEnvelope struct {
	Type string      `json:"type"`
	Data EventResult `json:"data"`
}

// EventResult contains the full result of an iRacing subsession.
type EventResult struct {
	SubsessionID            int                        `json:"subsession_id"`
	AssociatedSubsessionIDs []int                      `json:"associated_subsession_ids"`
	CanProtest              bool                       `json:"can_protest"`
	CarClasses              []CarClass                 `json:"car_classes"`
	CautionType             int                        `json:"caution_type"`
	CooldownMinutes         int                        `json:"cooldown_minutes"`
	CornersPerLap           int                        `json:"corners_per_lap"`
	DamageModel             int                        `json:"damage_model"`
	DriverChangeParam1      int                        `json:"driver_change_param1"`
	DriverChangeParam2      int                        `json:"driver_change_param2"`
	DriverChangeRule        int                        `json:"driver_change_rule"`
	DriverChanges           bool                       `json:"driver_changes"`
	DriverLicenses          map[string][]DriverLicense `json:"driver_licenses"`
	EndTime                 time.Time                  `json:"end_time"`
	EventAverageLap         int                        `json:"event_average_lap"`
	EventBestLapTime        int                        `json:"event_best_lap_time"`
	EventLapsComplete       int                        `json:"event_laps_complete"`
	EventStrengthOfField    int                        `json:"event_strength_of_field"`
	EventType               int                        `json:"event_type"`
	EventTypeName           string                     `json:"event_type_name"`
	HeatInfoID              int                        `json:"heat_info_id"`
	HostID                  int                        `json:"host_id"`
	LeagueID                int                        `json:"league_id"`
	LeagueSeasonID          int                        `json:"league_season_id"`
	LicenseCategory         string                     `json:"license_category"`
	LicenseCategoryID       int                        `json:"license_category_id"`
	LimitMinutes            int                        `json:"limit_minutes"`
	MaxTeamDrivers          int                        `json:"max_team_drivers"`
	MaxWeeks                int                        `json:"max_weeks"`
	MinTeamDrivers          int                        `json:"min_team_drivers"`
	NumCautionLaps          int                        `json:"num_caution_laps"`
	NumCautions             int                        `json:"num_cautions"`
	NumDrivers              int                        `json:"num_drivers"`
	NumLapsForQualAverage   int                        `json:"num_laps_for_qual_average"`
	NumLapsForSoloAverage   int                        `json:"num_laps_for_solo_average"`
	NumLeadChanges          int                        `json:"num_lead_changes"`
	OfficialSession         bool                       `json:"official_session"`
	PointsType              string                     `json:"points_type"`
	PrivateSessionID        int                        `json:"private_session_id"`
	RaceSummary             RaceSummary                `json:"race_summary"`
	RaceWeekNum             int                        `json:"race_week_num"`
	RestrictResults         bool                       `json:"restrict_results"`
	ResultsRestricted       bool                       `json:"results_restricted"`
	SeasonID                int                        `json:"season_id"`
	SeasonName              string                     `json:"season_name"`
	SeasonQuarter           int                        `json:"season_quarter"`
	SeasonShortName         string                     `json:"season_short_name"`
	SeasonYear              int                        `json:"season_year"`
	SeriesID                int                        `json:"series_id"`
	SeriesName              string                     `json:"series_name"`
	SeriesShortName         string                     `json:"series_short_name"`
	SessionID               int                        `json:"session_id"`
	SessionName             string                     `json:"session_name"`
	SessionResults          []SimSession               `json:"session_results"`
	SessionSplits           []SessionSplit             `json:"session_splits"`
	SpecialEventType        int                        `json:"special_event_type"`
	StartTime               time.Time                  `json:"start_time"`
	Track                   Track                      `json:"track"`
	TrackState              TrackState                 `json:"track_state"`
	Weather                 Weather                    `json:"weather"`
}

// CarClass describes a class of cars participating in an event.
type CarClass struct {
	CarClassID      int          `json:"car_class_id"`
	ShortName       string       `json:"short_name"`
	Name            string       `json:"name"`
	StrengthOfField int          `json:"strength_of_field"`
	NumEntries      int          `json:"num_entries"`
	CarsInClass     []CarInClass `json:"cars_in_class"`
}

// CarInClass is a car reference within a CarClass.
type CarInClass struct {
	CarID int `json:"car_id"`
}

// DriverLicense holds license and rating information for a single category.
type DriverLicense struct {
	CategoryID    int     `json:"category_id"`
	Category      string  `json:"category"`
	CategoryName  string  `json:"category_name"`
	LicenseLevel  int     `json:"license_level"`
	SafetyRating  float64 `json:"safety_rating"`
	CPI           float64 `json:"cpi"`
	IRating       int     `json:"irating"`
	TTRating      int     `json:"tt_rating"`
	MprNumRaces   int     `json:"mpr_num_races"`
	Color         string  `json:"color"`
	GroupName     string  `json:"group_name"`
	GroupID       int     `json:"group_id"`
	ProPromotable bool    `json:"pro_promotable"`
	Seq           int     `json:"seq"`
	MprNumTTs     int     `json:"mpr_num_tts"`
}

// RaceSummary holds aggregate statistics for the race subsession.
type RaceSummary struct {
	SubsessionID         int    `json:"subsession_id"`
	AverageLap           int    `json:"average_lap"`
	LapsComplete         int    `json:"laps_complete"`
	NumCautions          int    `json:"num_cautions"`
	NumCautionLaps       int    `json:"num_caution_laps"`
	NumLeadChanges       int    `json:"num_lead_changes"`
	FieldStrength        int    `json:"field_strength"`
	HeatInfoID           int    `json:"heat_info_id"`
	NumOptLaps           int    `json:"num_opt_laps"`
	HasOptPath           bool   `json:"has_opt_path"`
	SpecialEventType     int    `json:"special_event_type"`
	SpecialEventTypeText string `json:"special_event_type_text"`
}

// SimSession holds the results for one simulation session (practice, qualifying, race).
type SimSession struct {
	SimsessionNumber   int           `json:"simsession_number"`
	SimsessionName     string        `json:"simsession_name"`
	SimsessionType     int           `json:"simsession_type"`
	SimsessionTypeName string        `json:"simsession_type_name"`
	SimsessionSubtype  int           `json:"simsession_subtype"`
	WeatherResult      WeatherResult `json:"weather_result"`
	Results            []Result      `json:"results"`
}

// WeatherResult contains observed weather statistics for a simulation session.
type WeatherResult struct {
	AvgSkies                 int     `json:"avg_skies"`
	AvgCloudCoverPct         float64 `json:"avg_cloud_cover_pct"`
	MinCloudCoverPct         float64 `json:"min_cloud_cover_pct"`
	MaxCloudCoverPct         float64 `json:"max_cloud_cover_pct"`
	TempUnits                int     `json:"temp_units"`
	AvgTemp                  float64 `json:"avg_temp"`
	MinTemp                  float64 `json:"min_temp"`
	MaxTemp                  float64 `json:"max_temp"`
	AvgRelHumidity           float64 `json:"avg_rel_humidity"`
	WindUnits                int     `json:"wind_units"`
	AvgWindSpeed             float64 `json:"avg_wind_speed"`
	MinWindSpeed             float64 `json:"min_wind_speed"`
	MaxWindSpeed             float64 `json:"max_wind_speed"`
	AvgWindDir               int     `json:"avg_wind_dir"`
	MaxFog                   int     `json:"max_fog"`
	FogTimePct               int     `json:"fog_time_pct"`
	PrecipTimePct            float64 `json:"precip_time_pct"`
	PrecipMM                 int     `json:"precip_mm"`
	PrecipMM2HrBeforeSession int     `json:"precip_mm2hr_before_session"`
	SimulatedStartTime       string  `json:"simulated_start_time"`
}

// Result holds the performance data for a single driver in a simulation session.
type Result struct {
	CustID                  int            `json:"cust_id,omitempty"`
	TeamID                  int            `json:"team_id,omitempty"`
	DisplayName             string         `json:"display_name"`
	AggregateChampPoints    int            `json:"aggregate_champ_points"`
	AI                      bool           `json:"ai"`
	AverageLap              int            `json:"average_lap"`
	BestLapNum              int            `json:"best_lap_num"`
	BestLapTime             int            `json:"best_lap_time"`
	BestNLapsNum            int            `json:"best_nlaps_num"`
	BestNLapsTime           int            `json:"best_nlaps_time"`
	BestQualLapAt           string         `json:"best_qual_lap_at"`
	BestQualLapNum          int            `json:"best_qual_lap_num"`
	BestQualLapTime         int            `json:"best_qual_lap_time"`
	CarClassID              int            `json:"car_class_id"`
	CarClassName            string         `json:"car_class_name"`
	CarClassShortName       string         `json:"car_class_short_name"`
	CarID                   int            `json:"car_id"`
	CarName                 string         `json:"car_name"`
	CarCfg                  int            `json:"carcfg"`
	ChampPoints             int            `json:"champ_points"`
	ClassInterval           int            `json:"class_interval"`
	CountryCode             string         `json:"country_code"`
	Division                int            `json:"division"`
	DropRace                bool           `json:"drop_race"`
	FinishPosition          int            `json:"finish_position"`
	FinishPositionInClass   int            `json:"finish_position_in_class"`
	FlairID                 int            `json:"flair_id"`
	FlairName               string         `json:"flair_name"`
	FlairShortname          string         `json:"flair_shortname"`
	Friend                  bool           `json:"friend"`
	Helmet                  Helmet         `json:"helmet"`
	Incidents               int            `json:"incidents"`
	Interval                int            `json:"interval"`
	LapsComplete            int            `json:"laps_complete"`
	LapsLead                int            `json:"laps_lead"`
	LeagueAggPoints         int            `json:"league_agg_points"`
	LeaguePoints            int            `json:"league_points"`
	LicenseChangeOval       int            `json:"license_change_oval"`
	LicenseChangeRoad       int            `json:"license_change_road"`
	Livery                  Livery         `json:"livery"`
	MaxPctFuelFill          int            `json:"max_pct_fuel_fill"`
	NewCPI                  float64        `json:"new_cpi"`
	NewLicenseLevel         int            `json:"new_license_level"`
	NewSubLevel             int            `json:"new_sub_level"`
	NewTTRating             int            `json:"new_ttrating"`
	NewIRating              int            `json:"newi_rating"`
	OldCPI                  float64        `json:"old_cpi"`
	OldLicenseLevel         int            `json:"old_license_level"`
	OldSubLevel             int            `json:"old_sub_level"`
	OldTTRating             int            `json:"old_ttrating"`
	OldIRating              int            `json:"oldi_rating"`
	OptLapsComplete         int            `json:"opt_laps_complete"`
	Position                int            `json:"position"`
	QualLapTime             int            `json:"qual_lap_time"`
	ReasonOut               string         `json:"reason_out"`
	ReasonOutID             int            `json:"reason_out_id"`
	StartingPosition        int            `json:"starting_position"`
	StartingPositionInClass int            `json:"starting_position_in_class"`
	Suit                    Suit           `json:"suit"`
	Watched                 bool           `json:"watched"`
	WeightPenaltyKg         int            `json:"weight_penalty_kg"`
	DriverResults           []DriverResult `json:"driver_results,omitempty"`
}

// DriverResult holds a single driver's contribution in a team result entry.
type DriverResult struct {
	CustID       int    `json:"cust_id"`
	DisplayName  string `json:"display_name"`
	BestLapTime  int    `json:"best_lap_time"`
	LapsComplete int    `json:"laps_complete"`
	Incidents    int    `json:"incidents"`
}

// Helmet describes the visual configuration of a driver's helmet.
type Helmet struct {
	Pattern    int    `json:"pattern"`
	Color1     string `json:"color1"`
	Color2     string `json:"color2"`
	Color3     string `json:"color3"`
	FaceType   int    `json:"face_type"`
	HelmetType int    `json:"helmet_type"`
}

// Livery describes the visual configuration of a car's livery.
type Livery struct {
	CarID        int    `json:"car_id"`
	Pattern      int    `json:"pattern"`
	Color1       string `json:"color1"`
	Color2       string `json:"color2"`
	Color3       string `json:"color3"`
	NumberFont   int    `json:"number_font"`
	NumberColor1 string `json:"number_color1"`
	NumberColor2 string `json:"number_color2"`
	NumberColor3 string `json:"number_color3"`
	NumberSlant  int    `json:"number_slant"`
	Sponsor1     int    `json:"sponsor1"`
	Sponsor2     int    `json:"sponsor2"`
	CarNumber    string `json:"car_number"`
	WheelColor   string `json:"wheel_color"`
	RimType      int    `json:"rim_type"`
}

// Suit describes the visual configuration of a driver's suit.
type Suit struct {
	Pattern int    `json:"pattern"`
	Color1  string `json:"color1"`
	Color2  string `json:"color2"`
	Color3  string `json:"color3"`
}

// SessionSplit identifies a split subsession and its strength of field.
type SessionSplit struct {
	SubsessionID         int `json:"subsession_id"`
	EventStrengthOfField int `json:"event_strength_of_field"`
}

// Track holds track identification and category information.
type Track struct {
	Category   string `json:"category"`
	CategoryID int    `json:"category_id"`
	ConfigName string `json:"config_name"`
	TrackID    int    `json:"track_id"`
	TrackName  string `json:"track_name"`
}

// TrackState holds the rubber/marbles state of the track at the start of a session.
type TrackState struct {
	LeaveMarbles   bool `json:"leave_marbles"`
	PracticeRubber int  `json:"practice_rubber"`
	QualifyRubber  int  `json:"qualify_rubber"`
	RaceRubber     int  `json:"race_rubber"`
	WarmupRubber   int  `json:"warmup_rubber"`
}

// Weather holds the weather configuration for the session.
type Weather struct {
	AllowFog                      bool    `json:"allow_fog"`
	Fog                           int     `json:"fog"`
	PrecipMM2HrBeforeFinalSession int     `json:"precip_mm2hr_before_final_session"`
	PrecipMMFinalSession          int     `json:"precip_mm_final_session"`
	PrecipOption                  int     `json:"precip_option"`
	PrecipTimePct                 float64 `json:"precip_time_pct"`
	RelHumidity                   int     `json:"rel_humidity"`
	SimulatedStartTime            string  `json:"simulated_start_time"`
	Skies                         int     `json:"skies"`
	TempUnits                     int     `json:"temp_units"`
	TempValue                     int     `json:"temp_value"`
	TimeOfDay                     int     `json:"time_of_day"`
	TrackWater                    int     `json:"track_water"`
	Type                          int     `json:"type"`
	Version                       int     `json:"version"`
	WeatherVarInitial             int     `json:"weather_var_initial"`
	WeatherVarOngoing             int     `json:"weather_var_ongoing"`
	WindDir                       int     `json:"wind_dir"`
	WindUnits                     int     `json:"wind_units"`
	WindValue                     int     `json:"wind_value"`
}
