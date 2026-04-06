//nolint:lll // test code can be verbose
package importer

import (
	"errors"
	"fmt"
	"testing"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/aarondl/opt/null"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

type resolverCall struct {
	id   string
	name string
}

type fakeEntityResolver struct {
	resolveTrack    func(simTrackID, simTrackName string) (uint32, error)
	resolveDriver   func(simDriverID, simDriverName string) (uint32, error)
	resolveCar      func(simCarID, simCarName string) (uint32, error)
	resolveCarClass func(ownCarID uint32) (uint32, error)
	resolveTeam     func(ownDriverID uint32) (uint32, error)

	trackCalls    []resolverCall
	driverCalls   []resolverCall
	carCalls      []resolverCall
	carClassCalls []resolverCall
	teamCalls     []resolverCall
}

func (f *fakeEntityResolver) ResolveTrack(simTrackID, simTrackName string) (uint32, error) {
	f.trackCalls = append(f.trackCalls, resolverCall{id: simTrackID, name: simTrackName})
	return f.resolveTrack(simTrackID, simTrackName)
}

func (f *fakeEntityResolver) ResolveDriver(simDriverID, simDriverName string) (uint32, error) {
	f.driverCalls = append(f.driverCalls, resolverCall{id: simDriverID, name: simDriverName})
	return f.resolveDriver(simDriverID, simDriverName)
}

func (f *fakeEntityResolver) ResolveCar(simCarID, simCarName string) (uint32, error) {
	f.carCalls = append(f.carCalls, resolverCall{id: simCarID, name: simCarName})
	return f.resolveCar(simCarID, simCarName)
}

func (f *fakeEntityResolver) ResolveCarClass(ownCarID uint32) (uint32, error) {
	f.carClassCalls = append(
		f.carClassCalls,
		resolverCall{id: fmt.Sprintf("%d", ownCarID), name: ""},
	)
	return f.resolveCarClass(ownCarID)
}

func (f *fakeEntityResolver) ResolveTeam(ownDriverID uint32) (uint32, error) {
	f.teamCalls = append(f.teamCalls, resolverCall{id: fmt.Sprintf("%d", ownDriverID), name: ""})
	return f.resolveTeam(ownDriverID)
}

func TestResolveInputRequiresEntityResolver(t *testing.T) {
	t.Parallel()

	resolver := NewResolver(nil, nil)

	result, err := resolver.ResolveInput(&ParsedImportPayload{})
	if err == nil {
		t.Fatal("expected error when entity resolver is nil")
	}
	if result != nil {
		t.Fatal("expected nil result when entity resolver is nil")
	}
}

//nolint:funlen,gocyclo // test code
func TestResolveInput(t *testing.T) {
	t.Parallel()
	epi := &processor.EventProcInfo{
		Season: &models.Season{
			IsTeamBased:  false,
			IsMulticlass: false,
		},
	}
	testCases := []struct {
		name            string
		input           ParsedImportPayload
		fake            *fakeEntityResolver
		wantTrackID     uint32
		wantEntry       *models.ResultEntry
		wantUnresolved  []*commonv1.UnresolvedMapping
		wantTrackCalls  []resolverCall
		wantDriverCalls []resolverCall
		wantCarCalls    []resolverCall
	}{
		{
			name: "all entities resolved",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Silverstone"},
				Results: []*ResultRow{{
					FinPos:   2,
					Laps:     34,
					DriverID: "sim-driver-1",
					Name:     "Driver One",
					CarID:    "sim-car-1",
					Car:      "GT3",
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack: func(_, _ string) (uint32, error) {
					return 88, nil
				},
				resolveDriver: func(_, _ string) (uint32, error) {
					return 101, nil
				},
				resolveCar: func(_, _ string) (uint32, error) {
					return 202, nil
				},
			},
			wantTrackID: 88,
			wantEntry: &models.ResultEntry{
				FinishPosition: 2,
				LapsCompleted:  34,
				RawDriverName:  null.From("Driver One"),
				RawCarName:     null.From("GT3"),
				Incidents:      null.From(int32(0)),
				State:          "normal",
				DriverID:       null.From(int32(101)),
				CarModelID:     null.From(int32(202)),
			},
			wantTrackCalls:  []resolverCall{{id: "Silverstone", name: "Silverstone"}},
			wantDriverCalls: []resolverCall{{id: "sim-driver-1", name: "Driver One"}},
			wantCarCalls:    []resolverCall{{id: "sim-car-1", name: "GT3"}},
		},
		{
			name: "driver unresolved marks entry and records unresolved mapping",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Spa"},
				Results: []*ResultRow{{
					FinPos:   6,
					Laps:     20,
					DriverID: "sim-driver-2",
					Name:     "Unknown Driver",
					CarID:    "sim-car-2",
					Car:      "LMP2",
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack: func(_, _ string) (uint32, error) {
					return 9, nil
				},
				resolveDriver: func(_, _ string) (uint32, error) {
					return 0, errors.New("driver not found")
				},
				resolveCar: func(_, _ string) (uint32, error) {
					return 303, nil
				},
			},
			wantTrackID: 9,
			wantEntry: &models.ResultEntry{
				FinishPosition: 6,
				LapsCompleted:  20,
				RawDriverName:  null.From("Unknown Driver"),
				RawCarName:     null.From("LMP2"),
				Incidents:      null.From(int32(0)),
				State:          "mapping_error",
				CarModelID:     null.From(int32(303)),
			},
			wantUnresolved: []*commonv1.UnresolvedMapping{{
				SourceValue: "sim-driver-2 (name: Unknown Driver)",
				MappingType: "driver",
			}},
			wantTrackCalls:  []resolverCall{{id: "Spa", name: "Spa"}},
			wantDriverCalls: []resolverCall{{id: "sim-driver-2", name: "Unknown Driver"}},
			wantCarCalls:    []resolverCall{{id: "sim-car-2", name: "LMP2"}},
		},
		{
			name: "car and track unresolved are reported",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Unknown Track"},
				Results: []*ResultRow{{
					FinPos:   1,
					Laps:     18,
					DriverID: "sim-driver-3",
					Name:     "Driver Three",
					CarID:    "sim-car-3",
					Car:      "Unknown Car",
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack: func(_, _ string) (uint32, error) {
					return 0, errors.New("track not found")
				},
				resolveDriver: func(_, _ string) (uint32, error) {
					return 404, nil
				},
				resolveCar: func(_, _ string) (uint32, error) {
					return 0, errors.New("car not found")
				},
			},
			wantTrackID: 0,
			wantEntry: &models.ResultEntry{
				FinishPosition: 1,
				LapsCompleted:  18,
				RawDriverName:  null.From("Driver Three"),
				RawCarName:     null.From("Unknown Car"),
				Incidents:      null.From(int32(0)),
				State:          "mapping_error",
				DriverID:       null.From(int32(404)),
			},
			wantUnresolved: []*commonv1.UnresolvedMapping{
				{
					SourceValue: "sim-car-3 (name: Unknown Car)",
					MappingType: "car",
				},
				{
					SourceValue: "Unknown Track (name: Unknown Track)",
					MappingType: "track",
				},
			},
			wantTrackCalls:  []resolverCall{{id: "Unknown Track", name: "Unknown Track"}},
			wantDriverCalls: []resolverCall{{id: "sim-driver-3", name: "Driver Three"}},
			wantCarCalls:    []resolverCall{{id: "sim-car-3", name: "Unknown Car"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolver := NewResolver(tc.fake, epi)

			result, err := resolver.ResolveInput(&tc.input)
			if err != nil {
				t.Fatalf("ResolveInput returned unexpected error: %v", err)
			}
			if result.TrackID != tc.wantTrackID {
				t.Fatalf("unexpected track id: got %d want %d", result.TrackID, tc.wantTrackID)
			}
			if len(result.Entries) != 1 {
				t.Fatalf("unexpected entries length: got %d want 1", len(result.Entries))
			}

			got := result.Entries[0]
			if got.FinishPosition != tc.wantEntry.FinishPosition {
				t.Fatalf(
					"unexpected finishing position: got %d want %d",
					got.FinishPosition,
					tc.wantEntry.FinishPosition,
				)
			}
			if got.LapsCompleted != tc.wantEntry.LapsCompleted {
				t.Fatalf(
					"unexpected completed laps: got %d want %d",
					got.LapsCompleted,
					tc.wantEntry.LapsCompleted,
				)
			}
			if got.State != tc.wantEntry.State {
				t.Fatalf("unexpected state: got %v want %v", got.State, tc.wantEntry.State)
			}
			if got.DriverID != tc.wantEntry.DriverID {
				t.Fatalf(
					"unexpected driver id: got %v want %v",
					got.DriverID,
					tc.wantEntry.DriverID,
				)
			}
			if got.CarModelID != tc.wantEntry.CarModelID {
				t.Fatalf(
					"unexpected car model id: got %v want %v",
					got.CarModelID,
					tc.wantEntry.CarModelID,
				)
			}
			if got.RawDriverName != tc.wantEntry.RawDriverName {
				t.Fatalf(
					"unexpected driver name: got %v want %v",
					got.RawDriverName,
					tc.wantEntry.RawDriverName,
				)
			}
			if got.RawCarName != tc.wantEntry.RawCarName {
				t.Fatalf(
					"unexpected car name: got %v want %v",
					got.RawCarName,
					tc.wantEntry.RawCarName,
				)
			}

			unresolved := result.Unmapped
			if len(unresolved) != len(tc.wantUnresolved) {
				t.Fatalf(
					"unexpected unresolved length: got %d want %d",
					len(unresolved),
					len(tc.wantUnresolved),
				)
			}
			for i := range unresolved {
				if unresolved[i].SourceValue != tc.wantUnresolved[i].SourceValue {
					t.Fatalf(
						"unexpected unresolved source at index %d: got %q want %q",
						i,
						unresolved[i].SourceValue,
						tc.wantUnresolved[i].SourceValue,
					)
				}
				if unresolved[i].MappingType != tc.wantUnresolved[i].MappingType {
					t.Fatalf(
						"unexpected unresolved mapping type at index %d: got %q want %q",
						i,
						unresolved[i].MappingType,
						tc.wantUnresolved[i].MappingType,
					)
				}
			}

			assertResolverCalls(t, "track", tc.fake.trackCalls, tc.wantTrackCalls)
			assertResolverCalls(t, "driver", tc.fake.driverCalls, tc.wantDriverCalls)
			assertResolverCalls(t, "car", tc.fake.carCalls, tc.wantCarCalls)
		})
	}
}

func assertResolverCalls(t *testing.T, kind string, got, want []resolverCall) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected %s resolver call count: got %d want %d", kind, len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf(
				"unexpected %s resolver call at index %d: got %+v want %+v",
				kind,
				i,
				got[i],
				want[i],
			)
		}
	}
}

//nolint:funlen,gocyclo // test code
func TestResolveInputTeamBased(t *testing.T) {
	t.Parallel()
	epi := &processor.EventProcInfo{
		Season: &models.Season{
			IsTeamBased:  true,
			IsMulticlass: false,
		},
	}
	testCases := []struct {
		name            string
		input           ParsedImportPayload
		fake            *fakeEntityResolver
		wantTrackID     uint32
		wantEntry       *models.ResultEntry
		wantUnresolved  []*commonv1.UnresolvedMapping
		wantTrackCalls  []resolverCall
		wantDriverCalls []resolverCall
		wantCarCalls    []resolverCall
		wantTeamCalls   []resolverCall
	}{
		{
			name: "team and car resolved",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Silverstone"},
				Results: []*ResultRow{{
					FinPos: 1,
					Laps:   30,
					Name:   "Team Alpha",
					CarID:  "sim-car-1",
					Car:    "GT3",
					TeamDrivers: []*TeamDriver{
						{DriverID: "sim-drv-1", Name: "Driver A"},
					},
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack:  func(_, _ string) (uint32, error) { return 88, nil },
				resolveDriver: func(_, _ string) (uint32, error) { return 500, nil },
				resolveTeam:   func(_ uint32) (uint32, error) { return 700, nil },
				resolveCar:    func(_, _ string) (uint32, error) { return 202, nil },
			},
			wantTrackID: 88,
			wantEntry: &models.ResultEntry{
				FinishPosition: 1,
				LapsCompleted:  30,
				RawTeamName:    null.From("Team Alpha"),
				RawCarName:     null.From("GT3"),
				Incidents:      null.From(int32(0)),
				State:          "normal",
				TeamID:         null.From(int32(700)),
				CarModelID:     null.From(int32(202)),
			},
			wantTrackCalls:  []resolverCall{{id: "Silverstone", name: "Silverstone"}},
			wantDriverCalls: []resolverCall{{id: "sim-drv-1", name: "Driver A"}},
			wantCarCalls:    []resolverCall{{id: "sim-car-1", name: "GT3"}},
			wantTeamCalls:   []resolverCall{{id: "500", name: ""}},
		},
		{
			name: "team unresolved marks entry and records unresolved mapping",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Spa"},
				Results: []*ResultRow{{
					FinPos: 3,
					Laps:   20,
					Name:   "Team Beta",
					CarID:  "sim-car-2",
					Car:    "LMP2",
					TeamDrivers: []*TeamDriver{
						{DriverID: "sim-drv-2", Name: "Driver B"},
					},
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack:  func(_, _ string) (uint32, error) { return 9, nil },
				resolveDriver: func(_, _ string) (uint32, error) { return 501, nil },
				resolveTeam:   func(_ uint32) (uint32, error) { return 0, errors.New("team not found") },
				resolveCar:    func(_, _ string) (uint32, error) { return 303, nil },
			},
			wantTrackID: 9,
			wantEntry: &models.ResultEntry{
				FinishPosition: 3,
				LapsCompleted:  20,
				RawTeamName:    null.From("Team Beta"),
				RawCarName:     null.From("LMP2"),
				Incidents:      null.From(int32(0)),
				State:          "mapping_error",
				CarModelID:     null.From(int32(303)),
			},
			wantUnresolved: []*commonv1.UnresolvedMapping{{
				SourceValue: "team with driver 501 (name: Team Beta)",
				MappingType: "team",
			}},
			wantTrackCalls:  []resolverCall{{id: "Spa", name: "Spa"}},
			wantDriverCalls: []resolverCall{{id: "sim-drv-2", name: "Driver B"}},
			wantCarCalls:    []resolverCall{{id: "sim-car-2", name: "LMP2"}},
			wantTeamCalls:   []resolverCall{{id: "501", name: ""}},
		},
		{
			name: "multiple team drivers all driver resolver calls are made",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Monza"},
				Results: []*ResultRow{{
					FinPos: 2,
					Laps:   15,
					Name:   "Team Gamma",
					CarID:  "sim-car-3",
					Car:    "GTE",
					TeamDrivers: []*TeamDriver{
						{DriverID: "sim-drv-3", Name: "Driver C"},
						{DriverID: "sim-drv-4", Name: "Driver D"},
					},
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack:  func(_, _ string) (uint32, error) { return 77, nil },
				resolveDriver: func(_, _ string) (uint32, error) { return 600, nil },
				resolveTeam:   func(_ uint32) (uint32, error) { return 800, nil },
				resolveCar:    func(_, _ string) (uint32, error) { return 404, nil },
			},
			wantTrackID: 77,
			wantEntry: &models.ResultEntry{
				FinishPosition: 2,
				LapsCompleted:  15,
				RawTeamName:    null.From("Team Gamma"),
				RawCarName:     null.From("GTE"),
				Incidents:      null.From(int32(0)),
				State:          "normal",
				TeamID:         null.From(int32(800)),
				CarModelID:     null.From(int32(404)),
			},
			wantTrackCalls: []resolverCall{{id: "Monza", name: "Monza"}},
			wantDriverCalls: []resolverCall{
				{id: "sim-drv-3", name: "Driver C"},
			},
			wantCarCalls:  []resolverCall{{id: "sim-car-3", name: "GTE"}},
			wantTeamCalls: []resolverCall{{id: "600", name: ""}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolver := NewResolver(tc.fake, epi)

			result, err := resolver.ResolveInput(&tc.input)
			if err != nil {
				t.Fatalf("ResolveInput returned unexpected error: %v", err)
			}
			if result.TrackID != tc.wantTrackID {
				t.Fatalf("unexpected track id: got %d want %d", result.TrackID, tc.wantTrackID)
			}
			if len(result.Entries) != 1 {
				t.Fatalf("unexpected entries length: got %d want 1", len(result.Entries))
			}

			got := result.Entries[0]
			if got.FinishPosition != tc.wantEntry.FinishPosition {
				t.Fatalf(
					"unexpected finishing position: got %d want %d",
					got.FinishPosition,
					tc.wantEntry.FinishPosition,
				)
			}
			if got.LapsCompleted != tc.wantEntry.LapsCompleted {
				t.Fatalf(
					"unexpected completed laps: got %d want %d",
					got.LapsCompleted,
					tc.wantEntry.LapsCompleted,
				)
			}
			if got.State != tc.wantEntry.State {
				t.Fatalf("unexpected state: got %v want %v", got.State, tc.wantEntry.State)
			}
			if got.TeamID != tc.wantEntry.TeamID {
				t.Fatalf("unexpected team id: got %v want %v", got.TeamID, tc.wantEntry.TeamID)
			}
			if got.CarModelID != tc.wantEntry.CarModelID {
				t.Fatalf(
					"unexpected car model id: got %v want %v",
					got.CarModelID,
					tc.wantEntry.CarModelID,
				)
			}
			if got.RawTeamName != tc.wantEntry.RawTeamName {
				t.Fatalf(
					"unexpected team name: got %v want %v",
					got.RawTeamName,
					tc.wantEntry.RawTeamName,
				)
			}
			if got.RawCarName != tc.wantEntry.RawCarName {
				t.Fatalf(
					"unexpected car name: got %v want %v",
					got.RawCarName,
					tc.wantEntry.RawCarName,
				)
			}

			unresolved := result.Unmapped
			if len(unresolved) != len(tc.wantUnresolved) {
				t.Fatalf(
					"unexpected unresolved length: got %d want %d",
					len(unresolved),
					len(tc.wantUnresolved),
				)
			}
			for i := range unresolved {
				if unresolved[i].SourceValue != tc.wantUnresolved[i].SourceValue {
					t.Fatalf(
						"unexpected unresolved source at index %d: got %q want %q",
						i,
						unresolved[i].SourceValue,
						tc.wantUnresolved[i].SourceValue,
					)
				}
				if unresolved[i].MappingType != tc.wantUnresolved[i].MappingType {
					t.Fatalf(
						"unexpected unresolved mapping type at index %d: got %q want %q",
						i,
						unresolved[i].MappingType,
						tc.wantUnresolved[i].MappingType,
					)
				}
			}

			assertResolverCalls(t, "track", tc.fake.trackCalls, tc.wantTrackCalls)
			assertResolverCalls(t, "driver", tc.fake.driverCalls, tc.wantDriverCalls)
			assertResolverCalls(t, "car", tc.fake.carCalls, tc.wantCarCalls)
			assertResolverCalls(t, "team", tc.fake.teamCalls, tc.wantTeamCalls)
		})
	}
}

//nolint:funlen // test code
func TestResolveInputMulticlass(t *testing.T) {
	t.Parallel()
	epi := &processor.EventProcInfo{
		Season: &models.Season{
			IsTeamBased:  false,
			IsMulticlass: true,
		},
	}
	testCases := []struct {
		name              string
		input             ParsedImportPayload
		fake              *fakeEntityResolver
		wantEntry         *models.ResultEntry
		wantUnresolved    []*commonv1.UnresolvedMapping
		wantCarCalls      []resolverCall
		wantCarClassCalls []resolverCall
	}{
		{
			name: "car class resolved",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Nurburgring"},
				Results: []*ResultRow{{
					FinPos:   1,
					Laps:     25,
					DriverID: "sim-drv-1",
					Name:     "Driver One",
					CarID:    "sim-car-1",
					Car:      "GT3",
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack:    func(_, _ string) (uint32, error) { return 11, nil },
				resolveDriver:   func(_, _ string) (uint32, error) { return 101, nil },
				resolveCar:      func(_, _ string) (uint32, error) { return 202, nil },
				resolveCarClass: func(_ uint32) (uint32, error) { return 55, nil },
			},
			wantEntry: &models.ResultEntry{
				FinishPosition: 1,
				LapsCompleted:  25,
				RawDriverName:  null.From("Driver One"),
				RawCarName:     null.From("GT3"),
				Incidents:      null.From(int32(0)),
				State:          "normal",
				DriverID:       null.From(int32(101)),
				CarModelID:     null.From(int32(202)),
				CarClassID:     null.From(int32(55)),
			},
			wantCarCalls:      []resolverCall{{id: "sim-car-1", name: "GT3"}},
			wantCarClassCalls: []resolverCall{{id: "202", name: ""}},
		},
		{
			name: "car class unresolved marks entry and records unresolved mapping",
			input: ParsedImportPayload{
				Session: SessionInfo{Track: "Le Mans"},
				Results: []*ResultRow{{
					FinPos:   2,
					Laps:     12,
					DriverID: "sim-drv-2",
					Name:     "Driver Two",
					CarID:    "sim-car-2",
					Car:      "LMP2",
				}},
			},
			fake: &fakeEntityResolver{
				resolveTrack:    func(_, _ string) (uint32, error) { return 22, nil },
				resolveDriver:   func(_, _ string) (uint32, error) { return 102, nil },
				resolveCar:      func(_, _ string) (uint32, error) { return 303, nil },
				resolveCarClass: func(_ uint32) (uint32, error) { return 0, errors.New("car class not found") },
			},
			wantEntry: &models.ResultEntry{
				FinishPosition: 2,
				LapsCompleted:  12,
				RawDriverName:  null.From("Driver Two"),
				RawCarName:     null.From("LMP2"),
				Incidents:      null.From(int32(0)),
				State:          "mapping_error",
				DriverID:       null.From(int32(102)),
				CarModelID:     null.From(int32(303)),
			},
			wantUnresolved: []*commonv1.UnresolvedMapping{{
				SourceValue: "car class for car 303 (name: LMP2)",
				MappingType: "car_class",
			}},
			wantCarCalls:      []resolverCall{{id: "sim-car-2", name: "LMP2"}},
			wantCarClassCalls: []resolverCall{{id: "303", name: ""}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolver := NewResolver(tc.fake, epi)

			result, err := resolver.ResolveInput(&tc.input)
			if err != nil {
				t.Fatalf("ResolveInput returned unexpected error: %v", err)
			}
			if len(result.Entries) != 1 {
				t.Fatalf("unexpected entries length: got %d want 1", len(result.Entries))
			}

			got := result.Entries[0]
			if got.FinishPosition != tc.wantEntry.FinishPosition {
				t.Fatalf(
					"unexpected finishing position: got %d want %d",
					got.FinishPosition,
					tc.wantEntry.FinishPosition,
				)
			}
			if got.LapsCompleted != tc.wantEntry.LapsCompleted {
				t.Fatalf(
					"unexpected completed laps: got %d want %d",
					got.LapsCompleted,
					tc.wantEntry.LapsCompleted,
				)
			}
			if got.State != tc.wantEntry.State {
				t.Fatalf("unexpected state: got %v want %v", got.State, tc.wantEntry.State)
			}
			if got.DriverID != tc.wantEntry.DriverID {
				t.Fatalf(
					"unexpected driver id: got %v want %v",
					got.DriverID,
					tc.wantEntry.DriverID,
				)
			}
			if got.CarModelID != tc.wantEntry.CarModelID {
				t.Fatalf(
					"unexpected car model id: got %v want %v",
					got.CarModelID,
					tc.wantEntry.CarModelID,
				)
			}
			if got.CarClassID != tc.wantEntry.CarClassID {
				t.Fatalf(
					"unexpected car class id: got %v want %v",
					got.CarClassID,
					tc.wantEntry.CarClassID,
				)
			}

			unresolved := result.Unmapped
			if len(unresolved) != len(tc.wantUnresolved) {
				t.Fatalf(
					"unexpected unresolved length: got %d want %d",
					len(unresolved),
					len(tc.wantUnresolved),
				)
			}
			for i := range unresolved {
				if unresolved[i].SourceValue != tc.wantUnresolved[i].SourceValue {
					t.Fatalf(
						"unexpected unresolved source at index %d: got %q want %q",
						i,
						unresolved[i].SourceValue,
						tc.wantUnresolved[i].SourceValue,
					)
				}
				if unresolved[i].MappingType != tc.wantUnresolved[i].MappingType {
					t.Fatalf(
						"unexpected unresolved mapping type at index %d: got %q want %q",
						i,
						unresolved[i].MappingType,
						tc.wantUnresolved[i].MappingType,
					)
				}
			}

			assertResolverCalls(t, "car", tc.fake.carCalls, tc.wantCarCalls)
			assertResolverCalls(t, "car_class", tc.fake.carClassCalls, tc.wantCarClassCalls)
		})
	}
}
