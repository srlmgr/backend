//nolint:funlen,lll,dupl // tests
package command

import (
	"context"
	"errors"
	"testing"
	"time"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/srlmgr/backend/authn"
	"github.com/srlmgr/backend/db/models"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func TestSetSeasonDriversSuccessReplacesEntries(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "GT3 Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	carModel := seedCarModel(t, repo, seedCarManufacturer(t, repo, "Porsche").ID, "Porsche")
	carModelVariant := seedCarModelVariant(t, repo, carModel.ID, "911 GT3 R")
	oldDriver := seedDriver(t, repo, "driver-old", "Old Driver")
	newDriver := seedDriver(t, repo, "driver-new", "New Driver")

	_, err := repo.Drivers().
		SeasonDrivers().
		Create(context.Background(), &models.SeasonDriverSetter{
			DriverID:          omit.From(oldDriver.ID),
			SeasonID:          omit.From(season.ID),
			CarModelVariantID: omit.From(carModelVariant.ID),
			CarNumber:         omit.From("9"),
			CreatedBy:         omit.From(testUserSeed),
			UpdatedBy:         omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed season driver: %v", err)
	}

	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})
	resp, err := svc.SetSeasonDrivers(ctx, connect.NewRequest(&v1.SetSeasonDriversRequest{
		SeasonId: uint32(season.ID),
		Drivers: []*v1.SetSeasonDriver{
			{
				DriverId:          uint32(newDriver.ID),
				CarModelVariantId: uint32(carModelVariant.ID),
				CarNumber:         "27",
			},
		},
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg == nil {
		t.Fatal("expected non-nil response message")
	}

	items, err := repo.Drivers().SeasonDrivers().LoadBySeasonID(context.Background(), season.ID)
	if err != nil {
		t.Fatalf("failed to load season drivers: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected season driver count: got %d want %d", len(items), 1)
	}
	if items[0].DriverID != newDriver.ID {
		t.Fatalf("unexpected driver id: got %d want %d", items[0].DriverID, newDriver.ID)
	}
	if items[0].CreatedBy != testUserEditor || items[0].UpdatedBy != testUserEditor {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			items[0].CreatedBy,
			items[0].UpdatedBy,
		)
	}
}

func TestAddSeasonDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Assetto Corsa Competizione")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "GT World Challenge")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	carModel := seedCarModel(t, repo, seedCarManufacturer(t, repo, "Ferrari").ID, "Ferrari")
	carModelVariant := seedCarModelVariant(t, repo, carModel.ID, "296 GT3")
	driver := seedDriver(t, repo, "driver-1", "Driver One")
	joinedAt := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.AddSeasonDriver(ctx, connect.NewRequest(&v1.AddSeasonDriverRequest{
		DriverId:          uint32(driver.ID),
		SeasonId:          uint32(season.ID),
		CarModelVariantId: uint32(carModelVariant.ID),
		CarNumber:         "12",
		JoinedAt:          timestamppb.New(joinedAt),
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg == nil {
		t.Fatal("expected non-nil response message")
	}

	items, err := repo.Drivers().SeasonDrivers().LoadBySeasonID(context.Background(), season.ID)
	if err != nil {
		t.Fatalf("failed to load season drivers: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected season driver count: got %d want %d", len(items), 1)
	}
	if items[0].DriverID != driver.ID {
		t.Fatalf("unexpected driver id: got %d want %d", items[0].DriverID, driver.ID)
	}
	if items[0].CarModelVariantID != carModelVariant.ID {
		t.Fatalf(
			"unexpected car model variant id: got %d want %d",
			items[0].CarModelVariantID,
			carModelVariant.ID,
		)
	}
	if items[0].CarNumber != "12" {
		t.Fatalf("unexpected car number: got %q want %q", items[0].CarNumber, "12")
	}
	if !items[0].JoinedAt.Equal(joinedAt) {
		t.Fatalf("unexpected joined_at: got %s want %s", items[0].JoinedAt, joinedAt)
	}
}

func TestAddSeasonDriverFailureInvalidCarModelID(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "iRacing")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "LMP2 Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	driver := seedDriver(t, repo, "driver-1", "Driver One")

	_, err := svc.AddSeasonDriver(
		context.Background(),
		connect.NewRequest(&v1.AddSeasonDriverRequest{
			DriverId:          uint32(driver.ID),
			SeasonId:          uint32(season.ID),
			CarModelVariantId: uint32(0), // invalid car model ID
			CarNumber:         "12",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInvalidArgument)
	}
}

func TestRemoveSeasonDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "Le Mans Ultimate")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "Hypercar Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	carModel := seedCarModel(t, repo, seedCarManufacturer(t, repo, "Toyota").ID, "Toyota")
	carModelVariant := seedCarModelVariant(t, repo, carModel.ID, "GR010")
	driver := seedDriver(t, repo, "driver-1", "Driver One")

	created, err := repo.Drivers().
		SeasonDrivers().
		Create(context.Background(), &models.SeasonDriverSetter{
			DriverID:          omit.From(driver.ID),
			SeasonID:          omit.From(season.ID),
			CarModelVariantID: omit.From(carModelVariant.ID),
			CarNumber:         omit.From("8"),
			CreatedBy:         omit.From(testUserSeed),
			UpdatedBy:         omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed season driver: %v", err)
	}

	leftAt := time.Now().UTC().Truncate(time.Second)
	resp, err := svc.RemoveSeasonDriver(
		context.Background(),
		connect.NewRequest(&v1.RemoveSeasonDriverRequest{
			SeasonId: uint32(season.ID),
			DriverId: uint32(driver.ID),
			LeftAt:   timestamppb.New(leftAt),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg == nil {
		t.Fatal("expected non-nil response message")
	}

	updated, err := repo.Drivers().SeasonDrivers().LoadByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("failed to load season driver after remove: %v", err)
	}
	if updated.LeftAt.IsNull() {
		t.Fatal("expected left_at to be set")
	}
	if got := updated.LeftAt.GetOrZero().UTC().Truncate(time.Second); !got.Equal(leftAt) {
		t.Fatalf("unexpected left_at value: got %s want %s", got, leftAt)
	}
}

func TestDeleteSeasonDriverSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	sim := seedSimulation(t, repo, "rFactor 2")
	ps := seedPointSystem(t, repo, "Standard Points")
	series := seedSeries(t, repo, sim.ID, "Formula Series")
	season := seedSeason(t, repo, series.ID, ps.ID, "Season 2025")
	carModel := seedCarModel(t, repo, seedCarManufacturer(t, repo, "Audi").ID, "Audi")
	carModelVariant := seedCarModelVariant(t, repo, carModel.ID, "R8 LMS")
	driver := seedDriver(t, repo, "driver-1", "Driver One")

	created, err := repo.Drivers().
		SeasonDrivers().
		Create(context.Background(), &models.SeasonDriverSetter{
			DriverID:          omit.From(driver.ID),
			SeasonID:          omit.From(season.ID),
			CarModelVariantID: omit.From(carModelVariant.ID),
			CarNumber:         omit.From("44"),
			CreatedBy:         omit.From(testUserSeed),
			UpdatedBy:         omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed season driver: %v", err)
	}

	resp, err := svc.DeleteSeasonDriver(
		context.Background(),
		connect.NewRequest(&v1.DeleteSeasonDriverRequest{Id: uint32(created.ID)}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg == nil {
		t.Fatal("expected non-nil response message")
	}

	_, err = repo.Drivers().SeasonDrivers().LoadByID(context.Background(), created.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteSeasonDriverFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteSeasonDriver(
		context.Background(),
		connect.NewRequest(&v1.DeleteSeasonDriverRequest{Id: 1}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeInternal)
	}
	if !errors.Is(err, txErr) {
		t.Fatalf("expected wrapped transaction error: %v", err)
	}
}
