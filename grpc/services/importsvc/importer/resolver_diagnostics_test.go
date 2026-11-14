//nolint:lll,whitespace // table-driven test cases
package importer

import (
	"errors"
	"fmt"
	"testing"
	"time"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"
	"github.com/aarondl/opt/null"
	"google.golang.org/protobuf/proto"

	"github.com/srlmgr/backend/db/models"
	"github.com/srlmgr/backend/grpc/services/conversion"
	"github.com/srlmgr/backend/grpc/services/importsvc/processor"
	"github.com/srlmgr/backend/repository"
)

func diagnosticResolver() *fakeEntityResolver {
	return &fakeEntityResolver{
		resolveTrack:    func(string, string) (uint32, error) { return 1, nil },
		resolveDriver:   func(string, string) (uint32, error) { return 10, nil },
		resolveCar:      func(string, string) (uint32, error) { return 20, nil },
		resolveTeam:     func(uint32) (uint32, error) { return 30, nil },
		resolveCarClass: func(uint32) (uint32, error) { return 40, nil },
		seasonDriver:    func(uint32) (*models.SeasonDriver, error) { return &models.SeasonDriver{CarNumber: "12"}, nil },
		teamCarNumber:   func(uint32) (string, error) { return "12", nil },
	}
}

func diagnosticInput() *ParsedImportPayload {
	return &ParsedImportPayload{
		Session: SessionInfo{Track: "Spa"},
		Results: []*ResultRow{{
			DriverID: "driver-id", Name: "Driver", CarID: "car-id", Car: "Car", CarNumber: "12",
		}},
	}
}

func diagnosticInfo() *processor.EventProcInfo {
	return &processor.EventProcInfo{
		Season: &models.Season{ID: 2},
		Series: &models.Series{SimulationID: 3},
		Event:  &models.Event{ID: 4, EventDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}
}

//nolint:funlen,gocyclo // table-driven mapping cases
func TestResolverTypedDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*fakeEntityResolver, *ParsedImportPayload, *processor.EventProcInfo)
		check func(*testing.T, *commonv1.UnresolvedMapping)
	}{
		{
			name: "missing driver",
			setup: func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
				f.resolveDriver = func(string, string) (uint32, error) { return 0, repository.ErrNotFound }
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetSourceEntityNotFound()
				if x == nil ||
					x.EntityType != commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_DRIVER ||
					!proto.Equal(x.Source, sourceEntity("driver-id", "Driver")) {
					t.Fatalf("unexpected missing driver: %v", m)
				}
			},
		},
		{
			name: "inactive season driver",
			setup: func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
				f.seasonDriver = func(uint32) (*models.SeasonDriver, error) { return nil, repository.ErrNotFound }
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetSeasonDriverNotActive()
				if x == nil || x.DriverId != 10 ||
					!proto.Equal(x.SourceDriver, sourceEntity("driver-id", "Driver")) {
					t.Fatalf("unexpected inactive driver: %v", m)
				}
			},
		},
		{
			name: "missing car",
			setup: func(f *fakeEntityResolver, _ *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsMulticlass = true
				f.resolveCar = func(string, string) (uint32, error) { return 0, repository.ErrNotFound }
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetSourceEntityNotFound()
				if x == nil || x.EntityType != commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_CAR ||
					!proto.Equal(x.Source, sourceEntity("car-id", "Car")) {
					t.Fatalf("unexpected missing car: %v", m)
				}
			},
		},
		{
			name: "season car",
			setup: func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
				f.validateVariant = func(uint32) error { return repository.ErrNotFound }
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetSeasonCarNotConfigured()
				if x == nil || x.CarModelVariantId != 20 ||
					!proto.Equal(x.SourceCar, sourceEntity("car-id", "Car")) {
					t.Fatalf("unexpected season car: %v", m)
				}
			},
		},
		{
			name: "car class",
			setup: func(f *fakeEntityResolver, _ *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsMulticlass = true
				f.resolveCarClass = func(uint32) (uint32, error) { return 0, repository.ErrNotFound }
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetCarClassNotConfigured()
				if x == nil || x.CarModelVariantId != 20 ||
					!proto.Equal(x.SourceCar, sourceEntity("car-id", "Car")) {
					t.Fatalf("unexpected car class: %v", m)
				}
			},
		},
		{
			name: "driver number",
			setup: func(_ *fakeEntityResolver, in *ParsedImportPayload, _ *processor.EventProcInfo) {
				in.Results[0].CarNumber = " 13 "
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetCarNumberMismatch()
				if x == nil || x.GetDriverId() != 10 || x.ImportedNumber != " 13 " ||
					x.ConfiguredNumber == nil || x.GetConfiguredNumber() != "12" {
					t.Fatalf("unexpected driver number: %v", m)
				}
			},
		},
		{
			name: "team number",
			setup: func(_ *fakeEntityResolver, in *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsTeamBased = true
				in.Results[0].TeamDrivers = []*TeamDriver{{DriverID: "driver-id", Name: "Driver"}}
				in.Results[0].CarNumber = "13"
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetCarNumberMismatch()
				if x == nil || x.GetTeamId() != 30 || x.ImportedNumber != "13" ||
					x.ConfiguredNumber == nil || x.GetConfiguredNumber() != "12" {
					t.Fatalf("unexpected team number: %v", m)
				}
			},
		},
		{
			name: "missing team drivers",
			setup: func(_ *fakeEntityResolver, _ *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsTeamBased = true
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				if x := m.GetTeamDriversMissing(); x == nil || x.SourceTeamName != "Driver" {
					t.Fatalf("unexpected missing drivers: %v", m)
				}
			},
		},
		{
			name: "missing team",
			setup: func(f *fakeEntityResolver, in *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsTeamBased = true
				in.Results[0].TeamDrivers = []*TeamDriver{{DriverID: "driver-id", Name: "Driver"}}
				f.resolveTeam = func(uint32) (uint32, error) { return 0, repository.ErrNotFound }
			},
			check: func(t *testing.T, m *commonv1.UnresolvedMapping) {
				t.Helper()
				x := m.GetTeamNotFound()
				if x == nil || x.SourceTeamName != "Driver" || len(x.DriversChecked) != 1 ||
					x.DriversChecked[0].DriverId != 10 ||
					!proto.Equal(
						x.DriversChecked[0].SourceDriver,
						sourceEntity("driver-id", "Driver"),
					) {
					t.Fatalf("unexpected missing team: %v", m)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
			tt.setup(f, input, info)
			result, err := NewResolver(f, info).ResolveInput(input)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Unmapped) != 1 ||
				result.Entries[0].State != conversion.ResultStateMappingError {
				t.Fatalf("unexpected mappings or state: %+v", result)
			}
			m := result.Unmapped[0]
			if m.ResultIndex == nil || m.GetResultIndex() != 0 || m.ResultEntryId != nil ||
				m.GetSimulationId() != 3 || m.GetSeasonId() != 2 || m.GetEventId() != 4 ||
				m.EventTime == nil || !m.EventTime.AsTime().Equal(info.Event.EventDate) {
				t.Fatalf("unexpected mapping context: %v", m)
			}
			tt.check(t, m)
			if tt.name == "missing car" && len(f.carClassCalls) != 0 {
				t.Fatal("class lookup with unresolved car")
			}
		})
	}
}

func TestResolverTrackAndMultipleTeamDrivers(t *testing.T) {
	f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
	info.Season.IsTeamBased = true
	input.Results[0].TeamDrivers = []*TeamDriver{
		{DriverID: "a", Name: "A"},
		{DriverID: "b", Name: "B"},
	}
	f.resolveDriver = func(string, string) (uint32, error) { return 0, repository.ErrNotFound }
	f.resolveTrack = func(string, string) (uint32, error) { return 0, repository.ErrNotFound }
	result, err := NewResolver(f, info).ResolveInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unmapped) != 3 {
		t.Fatalf("expected two drivers and a track, got %v", result.Unmapped)
	}
	for i, id := range []string{"a", "b"} {
		x := result.Unmapped[i].GetSourceEntityNotFound()
		if x == nil || x.GetSource().GetId() != id || result.Unmapped[i].ResultIndex == nil {
			t.Fatalf("unexpected driver diagnostic %d: %v", i, result.Unmapped[i])
		}
	}
	track := result.Unmapped[2]
	x := track.GetSourceEntityNotFound()
	if x == nil || x.EntityType != commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_TRACK ||
		x.Source.Id != nil || x.Source.GetName() != "Spa" || track.ResultIndex != nil ||
		track.ResultEntryId != nil {
		t.Fatalf("unexpected track diagnostic: %v", track)
	}
}

//nolint:funlen // table-driven failure cases
func TestResolverPropagatesOperationalErrors(t *testing.T) {
	failure := errors.New("database unavailable")
	tests := []struct {
		name  string
		setup func(*fakeEntityResolver, *ParsedImportPayload, *processor.EventProcInfo)
	}{
		{"driver", func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
			f.resolveDriver = func(string, string) (uint32, error) { return 0, failure }
		}},
		{
			"season driver",
			func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
				f.seasonDriver = func(uint32) (*models.SeasonDriver, error) { return nil, failure }
			},
		},
		{"car", func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
			f.resolveCar = func(string, string) (uint32, error) { return 0, failure }
		}},
		{
			"season car",
			func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
				f.validateVariant = func(uint32) error { return failure }
			},
		},
		{
			"class",
			func(f *fakeEntityResolver, _ *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsMulticlass = true
				f.resolveCarClass = func(uint32) (uint32, error) { return 0, failure }
			},
		},
		{
			"team",
			func(f *fakeEntityResolver, in *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsTeamBased = true
				in.Results[0].TeamDrivers = []*TeamDriver{{Name: "Driver"}}
				f.resolveTeam = func(uint32) (uint32, error) { return 0, failure }
			},
		},
		{
			"team number",
			func(f *fakeEntityResolver, in *ParsedImportPayload, info *processor.EventProcInfo) {
				info.Season.IsTeamBased = true
				in.Results[0].TeamDrivers = []*TeamDriver{{Name: "Driver"}}
				f.teamCarNumber = func(uint32) (string, error) { return "", failure }
			},
		},
		{"track", func(f *fakeEntityResolver, _ *ParsedImportPayload, _ *processor.EventProcInfo) {
			f.resolveTrack = func(string, string) (uint32, error) { return 0, failure }
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
			tt.setup(f, input, info)
			result, err := NewResolver(f, info).ResolveInput(input)
			if result != nil || !errors.Is(err, failure) {
				t.Fatalf("expected operational failure, got result=%v err=%v", result, err)
			}
		})
	}
}

//nolint:funlen // retry fixture and assertions
func TestResolveNonMappedMatchesRowsWithoutOrdering(t *testing.T) {
	f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
	input.Results = append(
		input.Results,
		&ResultRow{
			DriverID:  "second-id",
			Name:      "Second",
			CarID:     "second-car",
			Car:       "Car2",
			CarNumber: "12",
		},
	)
	f.resolveDriver = func(id, _ string) (uint32, error) {
		if id == "second-id" {
			return 0, repository.ErrNotFound
		}
		return 10, nil
	}
	existing := []*models.ResultEntry{
		{
			ID:            9,
			RaceGridID:    5,
			State:         conversion.ResultStateMappingError,
			RawDriverName: null.From("Second"),
			RawCarName:    null.From("Car2"),
			CarNumber:     null.From("12"),
			DriverID:      null.From(int32(99)),
		},
		{
			ID:            8,
			RaceGridID:    5,
			State:         conversion.ResultStateMappingError,
			RawDriverName: null.From("Driver"),
			RawCarName:    null.From("Car"),
			CarNumber:     null.From("12"),
		},
	}
	result, err := NewResolver(f, info).ResolveNonMapped(input, existing)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Unmapped) != 1 || result.Unmapped[0].GetResultIndex() != 1 ||
		result.Unmapped[0].GetResultEntryId() != 9 || result.Unmapped[0].GetRaceGridId() != 5 ||
		result.Unmapped[0].GetSourceEntityNotFound().GetSource().GetId() != "second-id" ||
		!result.Entries[0].DriverID.IsNull() || result.Entries[1].State != conversion.ResultStateNormal {
		t.Fatalf("unexpected retry result: %+v, diagnostics %v", result.Entries, result.Unmapped)
	}
	if existing[0].DriverID.IsNull() {
		t.Fatal("retry mutated existing entry")
	}
}

func TestResolveNonMappedRejectsAmbiguousRows(t *testing.T) {
	f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
	input.Results = append(input.Results, input.Results[0])
	entry := &models.ResultEntry{
		ID:    9,
		State: conversion.ResultStateMappingError,
		RawDriverName: null.From(
			"Driver",
		),
		RawCarName: null.From("Car"),
		CarNumber:  null.From("12"),
	}
	if _, err := NewResolver(
		f,
		info,
	).ResolveNonMapped(input, []*models.ResultEntry{entry}); err == nil {
		t.Fatal("expected ambiguous row error")
	}
}

func TestResolveNonMappedRejectsMissingRow(t *testing.T) {
	f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
	entry := &models.ResultEntry{
		ID: 9, State: conversion.ResultStateMappingError,
		RawDriverName: null.From("Someone else"), RawCarName: null.From("Car"),
		CarNumber: null.From("12"),
	}
	if _, err := NewResolver(
		f,
		info,
	).ResolveNonMapped(input, []*models.ResultEntry{entry}); err == nil {
		t.Fatal("expected missing row error")
	}
}

func TestTrackErrorDoesNotMarkEntries(t *testing.T) {
	f, input, info := diagnosticResolver(), diagnosticInput(), diagnosticInfo()
	f.resolveTrack = func(string, string) (uint32, error) { return 0, repository.ErrNotFound }
	result, err := NewResolver(f, info).ResolveInput(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Entries[0].State != conversion.ResultStateNormal || len(result.Unmapped) != 1 ||
		result.Unmapped[0].GetSourceEntityNotFound().EntityType != commonv1.SourceEntityType_SOURCE_ENTITY_TYPE_TRACK {
		t.Fatalf("unexpected track result: %+v", result)
	}
}

func TestRepositoryResolveByPreservesNotFound(t *testing.T) {
	r := &RepositoryEntityResolver{sim: &models.RacingSim{Name: "sim"}}
	_, err := r.resolveBy("driver", "id", "name", func(string) (uint32, error) {
		return 0, fmt.Errorf("lookup: %w", repository.ErrNotFound)
	})
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
