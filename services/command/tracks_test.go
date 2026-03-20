//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package command

import (
	"context"
	"errors"
	"testing"

	v1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/command/v1"
	"connectrpc.com/connect"

	"github.com/srlmgr/backend/authn"
	postgresrepo "github.com/srlmgr/backend/repository/postgres"
	"github.com/srlmgr/backend/repository/repoerrors"
)

func TestTrackSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (trackSetterBuilder{}).Build(&v1.CreateTrackRequest{
		Name:       "Spa-Francorchamps",
		Country:    "Belgium",
		Latitude:   50.4372,
		Longitude:  5.9714,
		WebsiteUrl: "https://www.spa-francorchamps.be",
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "Spa-Francorchamps" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if setter.Country.IsUnset() {
		t.Fatal("expected country to be set")
	}
	if country := setter.Country.MustGetNull().GetOr(""); country != "Belgium" {
		t.Fatalf("unexpected country setter value: %q", country)
	}
	if setter.Latitude.IsUnset() {
		t.Fatal("expected latitude to be set")
	}
	if setter.Longitude.IsUnset() {
		t.Fatal("expected longitude to be set")
	}
	if setter.WebsiteURL.IsUnset() {
		t.Fatal("expected website_url to be set")
	}
	if url := setter.WebsiteURL.MustGetNull().GetOr(""); url != "https://www.spa-francorchamps.be" {
		t.Fatalf("unexpected website_url setter value: %q", url)
	}
}

func TestTrackSetterBuilderBuildOptionalFieldsNotSet(t *testing.T) {
	t.Parallel()

	setter := (trackSetterBuilder{}).Build(&v1.CreateTrackRequest{
		Name: "Monza",
	})

	if !setter.Name.IsValue() || setter.Name.MustGet() != "Monza" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.Country.IsUnset() {
		t.Fatal("expected country to be unset when zero value")
	}
	if !setter.Latitude.IsUnset() {
		t.Fatal("expected latitude to be unset when zero value")
	}
	if !setter.Longitude.IsUnset() {
		t.Fatal("expected longitude to be unset when zero value")
	}
	if !setter.WebsiteURL.IsUnset() {
		t.Fatal("expected website_url to be unset when zero value")
	}
}

func TestTrackLayoutSetterBuilderBuildSuccess(t *testing.T) {
	t.Parallel()

	setter := (trackLayoutSetterBuilder{}).Build(&v1.CreateTrackLayoutRequest{
		TrackId:        7,
		Name:           "Grand Prix",
		LengthMeters:   7004,
		LayoutImageUrl: "https://example.com/spa-gp.png",
	})

	if !setter.TrackID.IsValue() || setter.TrackID.MustGet() != 7 {
		t.Fatalf("unexpected track_id setter value: %+v", setter.TrackID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "Grand Prix" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if setter.LengthMeters.IsUnset() {
		t.Fatal("expected length_meters to be set")
	}
	if lm := setter.LengthMeters.MustGetNull().GetOr(0); lm != 7004 {
		t.Fatalf("unexpected length_meters setter value: %d", lm)
	}
	if setter.LayoutImageURL.IsUnset() {
		t.Fatal("expected layout_image_url to be set")
	}
	if url := setter.LayoutImageURL.MustGetNull().GetOr(""); url != "https://example.com/spa-gp.png" {
		t.Fatalf("unexpected layout_image_url setter value: %q", url)
	}
}

func TestTrackLayoutSetterBuilderBuildOptionalFieldsNotSet(t *testing.T) {
	t.Parallel()

	setter := (trackLayoutSetterBuilder{}).Build(&v1.CreateTrackLayoutRequest{
		TrackId: 3,
		Name:    "National",
	})

	if !setter.TrackID.IsValue() || setter.TrackID.MustGet() != 3 {
		t.Fatalf("unexpected track_id setter value: %+v", setter.TrackID)
	}
	if !setter.Name.IsValue() || setter.Name.MustGet() != "National" {
		t.Fatalf("unexpected name setter value: %+v", setter.Name)
	}
	if !setter.LengthMeters.IsUnset() {
		t.Fatal("expected length_meters to be unset when zero value")
	}
	if !setter.LayoutImageURL.IsUnset() {
		t.Fatal("expected layout_image_url to be unset when zero value")
	}
}

func TestCreateTrackSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateTrack(ctx, connect.NewRequest(&v1.CreateTrackRequest{
		Name:    "Silverstone",
		Country: "United Kingdom",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTrack().GetName() != "Silverstone" {
		t.Fatalf("unexpected track name: %q", resp.Msg.GetTrack().GetName())
	}
	if resp.Msg.GetTrack().GetCountry() != "United Kingdom" {
		t.Fatalf("unexpected country: %q", resp.Msg.GetTrack().GetCountry())
	}

	id := int32(resp.Msg.GetTrack().GetId())
	stored, err := repo.Tracks().Tracks().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created track: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
}

func TestCreateTrackFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedTrack(t, repo, "Nürburgring")

	_, err := svc.CreateTrack(
		context.Background(),
		connect.NewRequest(&v1.CreateTrackRequest{
			Name: "Nürburgring",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestCreateTrackFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.CreateTrack(
		context.Background(),
		connect.NewRequest(&v1.CreateTrackRequest{
			Name: "Zandvoort",
		}),
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

func TestUpdateTrackSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedTrack(t, repo, "Brands Hatch")
	before, err := repo.Tracks().Tracks().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial track: %v", err)
	}

	resp, err := svc.UpdateTrack(ctx, connect.NewRequest(&v1.UpdateTrackRequest{
		TrackId: uint32(initial.ID),
		Name:    "Brands Hatch Updated",
		Country: "United Kingdom",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTrack().GetName() != "Brands Hatch Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetTrack().GetName())
	}

	after, err := repo.Tracks().Tracks().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated track: %v", err)
	}
	if after.UpdatedBy != testUserEditor {
		t.Fatalf("unexpected UpdatedBy: got %q want %q", after.UpdatedBy, testUserEditor)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected UpdatedAt to move forward: before=%s after=%s",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
}

func TestUpdateTrackFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateTrack(
		context.Background(),
		connect.NewRequest(&v1.UpdateTrackRequest{
			TrackId: 999,
			Name:    "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestUpdateTrackFailureDuplicateName(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	seedTrack(t, repo, "Circuit de la Sarthe")
	second := seedTrack(t, repo, "Laguna Seca")

	_, err := svc.UpdateTrack(
		context.Background(),
		connect.NewRequest(&v1.UpdateTrackRequest{
			TrackId: uint32(second.ID),
			Name:    "Circuit de la Sarthe",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate update error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}

	stored, loadErr := repo.Tracks().Tracks().LoadByID(context.Background(), second.ID)
	if loadErr != nil {
		t.Fatalf("failed to load track after failed update: %v", loadErr)
	}
	if stored.Name != "Laguna Seca" {
		t.Fatalf(
			"unexpected name after failed duplicate update: got %q want %q",
			stored.Name,
			"Laguna Seca",
		)
	}
}

func TestDeleteTrackSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	initial := seedTrack(t, repo, "Delete Me Track")

	resp, err := svc.DeleteTrack(
		context.Background(),
		connect.NewRequest(&v1.DeleteTrackRequest{
			TrackId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Tracks().Tracks().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestDeleteTrackFailureTransactionError(t *testing.T) {
	repo := postgresrepo.New(testPool)
	txErr := errors.New(txFailedErrMsg)
	svc := newTestService(repo, txManagerStub{
		runInTx: func(_ context.Context, _ func(ctx context.Context) error) error {
			return txErr
		},
	})

	_, err := svc.DeleteTrack(
		context.Background(),
		connect.NewRequest(&v1.DeleteTrackRequest{
			TrackId: 1,
		}),
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

func TestCreateTrackLayoutSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	track := seedTrack(t, repo, "Spa-Francorchamps")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserTester})

	resp, err := svc.CreateTrackLayout(ctx, connect.NewRequest(&v1.CreateTrackLayoutRequest{
		TrackId:      uint32(track.ID),
		Name:         "Grand Prix",
		LengthMeters: 7004,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTrackLayout().GetName() != "Grand Prix" {
		t.Fatalf("unexpected layout name: %q", resp.Msg.GetTrackLayout().GetName())
	}
	if resp.Msg.GetTrackLayout().GetTrackId() != uint32(track.ID) {
		t.Fatalf(
			"unexpected track_id: got %d want %d",
			resp.Msg.GetTrackLayout().GetTrackId(),
			track.ID,
		)
	}
	if resp.Msg.GetTrackLayout().GetLengthMeters() != 7004 {
		t.Fatalf("unexpected length_meters: %d", resp.Msg.GetTrackLayout().GetLengthMeters())
	}

	id := int32(resp.Msg.GetTrackLayout().GetId())
	stored, err := repo.Tracks().TrackLayouts().LoadByID(context.Background(), id)
	if err != nil {
		t.Fatalf("failed to load created track layout: %v", err)
	}
	if stored.CreatedBy != testUserTester || stored.UpdatedBy != testUserTester {
		t.Fatalf(
			"unexpected created/updated by values: %q / %q",
			stored.CreatedBy,
			stored.UpdatedBy,
		)
	}
	if stored.TrackID != track.ID {
		t.Fatalf("unexpected stored track_id: got %d want %d", stored.TrackID, track.ID)
	}
}

func TestCreateTrackLayoutFailureDuplicateNameSameTrack(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	track := seedTrack(t, repo, "Monza")
	seedTrackLayout(t, repo, track.ID, "Grand Prix")

	_, err := svc.CreateTrackLayout(
		context.Background(),
		connect.NewRequest(&v1.CreateTrackLayoutRequest{
			TrackId: uint32(track.ID),
			Name:    "Grand Prix",
		}),
	)
	if err == nil {
		t.Fatal("expected duplicate create error")
	}
	if got := connect.CodeOf(err); got != connect.CodeAlreadyExists {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeAlreadyExists)
	}
}

func TestUpdateTrackLayoutSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	track := seedTrack(t, repo, "Nürburgring")
	ctx := authn.AddPrincipal(context.Background(), &authn.Principal{Name: testUserEditor})

	initial := seedTrackLayout(t, repo, track.ID, "GP")
	before, err := repo.Tracks().TrackLayouts().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load initial track layout: %v", err)
	}

	resp, err := svc.UpdateTrackLayout(ctx, connect.NewRequest(&v1.UpdateTrackLayoutRequest{
		TrackLayoutId: uint32(initial.ID),
		TrackId:       uint32(track.ID),
		Name:          "Grand Prix Updated",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.GetTrackLayout().GetName() != "Grand Prix Updated" {
		t.Fatalf("unexpected updated name: %q", resp.Msg.GetTrackLayout().GetName())
	}

	after, err := repo.Tracks().TrackLayouts().LoadByID(context.Background(), initial.ID)
	if err != nil {
		t.Fatalf("failed to load updated track layout: %v", err)
	}
	if after.UpdatedBy != testUserEditor {
		t.Fatalf("unexpected UpdatedBy: got %q want %q", after.UpdatedBy, testUserEditor)
	}
	if !after.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf(
			"expected UpdatedAt to move forward: before=%s after=%s",
			before.UpdatedAt,
			after.UpdatedAt,
		)
	}
}

func TestUpdateTrackLayoutFailureNotFound(t *testing.T) {
	svc, _ := newDBBackedTestService(t)

	_, err := svc.UpdateTrackLayout(
		context.Background(),
		connect.NewRequest(&v1.UpdateTrackLayoutRequest{
			TrackLayoutId: 999,
			Name:          "missing",
		}),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Fatalf("unexpected code: got %v want %v", got, connect.CodeNotFound)
	}
}

func TestDeleteTrackLayoutSuccess(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	track := seedTrack(t, repo, "Interlagos")
	initial := seedTrackLayout(t, repo, track.ID, "Full Circuit")

	resp, err := svc.DeleteTrackLayout(
		context.Background(),
		connect.NewRequest(&v1.DeleteTrackLayoutRequest{
			TrackLayoutId: uint32(initial.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}

	_, err = repo.Tracks().TrackLayouts().LoadByID(context.Background(), initial.ID)
	if !errors.Is(err, repoerrors.ErrNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestCreateTrackLayoutSuccessDuplicateNameDifferentTrack(t *testing.T) {
	svc, repo := newDBBackedTestService(t)
	firstTrack := seedTrack(t, repo, "Silverstone A")
	secondTrack := seedTrack(t, repo, "Silverstone B")
	seedTrackLayout(t, repo, firstTrack.ID, "International")

	resp, err := svc.CreateTrackLayout(
		context.Background(),
		connect.NewRequest(&v1.CreateTrackLayoutRequest{
			TrackId: uint32(secondTrack.ID),
			Name:    "International",
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error for same name on different track: %v", err)
	}
	if resp.Msg.GetTrackLayout().GetTrackId() != uint32(secondTrack.ID) {
		t.Fatalf(
			"unexpected track_id: got %d want %d",
			resp.Msg.GetTrackLayout().GetTrackId(),
			secondTrack.ID,
		)
	}
}
