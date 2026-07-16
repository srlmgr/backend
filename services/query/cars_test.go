//nolint:lll,dupl // test files can have some duplication and long lines for test data setup
package query

import (
	"context"
	"errors"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
	"github.com/aarondl/opt/omit"

	"github.com/srlmgr/backend/db/models"
	rootrepo "github.com/srlmgr/backend/repository"
)

//nolint:whitespace // multiline signature style
func seedCarManufacturer(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) *models.CarManufacturer {
	t.Helper()
	m, err := repo.Cars().
		CarManufacturers().
		Create(context.Background(), &models.CarManufacturerSetter{
			Name:      omit.From(name),
			CreatedBy: omit.From(testUserSeed),
			UpdatedBy: omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed car manufacturer %q: %v", name, err)
	}
	return m
}

//nolint:whitespace // multiline signature style
func seedCarModelV2(
	t *testing.T,
	repo rootrepo.Repository,
	manufacturerID int32,
	name string,
) *models.CarModel {
	t.Helper()
	cm, err := repo.Cars().CarModels().Create(context.Background(), &models.CarModelSetter{
		ManufacturerID: omit.From(manufacturerID),
		Name:           omit.From(name),
		IsActive:       omit.From(true),
		CreatedBy:      omit.From(testUserSeed),
		UpdatedBy:      omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car model v2 %q: %v", name, err)
	}
	return cm
}

//nolint:whitespace // multiline signature style
func seedCarModelVariant(
	t *testing.T,
	repo rootrepo.Repository,
	carModelV2ID int32,
	name string,
) *models.CarModelVariant {
	t.Helper()
	variant, err := repo.Cars().
		CarModelVariants().
		Create(context.Background(), &models.CarModelVariantSetter{
			CarModelID: omit.From(carModelV2ID),
			Name:       omit.From(name),
			IsActive:   omit.From(true),
			CreatedBy:  omit.From(testUserSeed),
			UpdatedBy:  omit.From(testUserSeed),
		})
	if err != nil {
		t.Fatalf("failed to seed car model variant %q: %v", name, err)
	}
	return variant
}

func TestListCarManufacturersEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListCarManufacturers(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarManufacturersRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListCarManufacturersReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	alpha := seedCarManufacturer(t, repo, "Alpha Motors")
	beta := seedCarManufacturer(t, repo, "Beta Motors")

	resp, err := svc.ListCarManufacturers(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarManufacturersRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
	}

	if !ids[uint32(alpha.ID)] {
		t.Errorf("alpha manufacturer (id=%d) not found in response", alpha.ID)
	}
	if !ids[uint32(beta.ID)] {
		t.Errorf("beta manufacturer (id=%d) not found in response", beta.ID)
	}
}

func TestGetCarManufacturerSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	m := seedCarManufacturer(t, repo, "Apex Motorsports")

	resp, err := svc.GetCarManufacturer(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarManufacturerRequest{
			Id: uint32(m.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetCarManufacturer().GetId() != uint32(m.ID) {
		t.Errorf("expected id %d, got %d", m.ID, resp.Msg.GetCarManufacturer().GetId())
	}
	if resp.Msg.GetCarManufacturer().GetName() != "Apex Motorsports" {
		t.Errorf(
			"expected name %q, got %q",
			"Apex Motorsports",
			resp.Msg.GetCarManufacturer().GetName(),
		)
	}
}

func TestGetCarManufacturerNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetCarManufacturer(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarManufacturerRequest{
			Id: 99999,
		}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestListCarModelsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListCarModels(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarModelsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListCarModelsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "All Models V2 Manufacturer")
	model1 := seedCarModelV2(t, repo, mfr.ID, "Model V2 X")
	model2 := seedCarModelV2(t, repo, mfr.ID, "Model V2 Y")

	resp, err := svc.ListCarModels(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarModelsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
	}

	if !ids[uint32(model1.ID)] {
		t.Errorf("model1 (id=%d) not found in response", model1.ID)
	}
	if !ids[uint32(model2.ID)] {
		t.Errorf("model2 (id=%d) not found in response", model2.ID)
	}
}

func TestListCarModelsByManufacturerID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr1 := seedCarManufacturer(t, repo, "Manufacturer V2 One")
	mfr2 := seedCarManufacturer(t, repo, "Manufacturer V2 Two")
	model1 := seedCarModelV2(t, repo, mfr1.ID, "Model V2 for Mfr One")
	_ = seedCarModelV2(t, repo, mfr2.ID, "Model V2 for Mfr Two")

	resp, err := svc.ListCarModels(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarModelsRequest{
			ManufacturerId: uint32(mfr1.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetId() != uint32(model1.ID) {
		t.Errorf("expected model id %d, got %d", model1.ID, items[0].GetId())
	}
	if items[0].GetManufacturerId() != uint32(mfr1.ID) {
		t.Errorf("expected manufacturer id %d, got %d", mfr1.ID, items[0].GetManufacturerId())
	}
}

func TestGetCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "Success V2 Manufacturer")
	cm := seedCarModelV2(t, repo, mfr.ID, "Success V2 Model")

	resp, err := svc.GetCarModel(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarModelRequest{
			Id: uint32(cm.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetCarModel().GetId() != uint32(cm.ID) {
		t.Errorf("expected id %d, got %d", cm.ID, resp.Msg.GetCarModel().GetId())
	}
	if resp.Msg.GetCarModel().GetManufacturerId() != uint32(mfr.ID) {
		t.Errorf(
			"expected manufacturer id %d, got %d",
			mfr.ID,
			resp.Msg.GetCarModel().GetManufacturerId(),
		)
	}
	if resp.Msg.GetCarModel().GetName() != "Success V2 Model" {
		t.Errorf("expected name %q, got %q", "Success V2 Model", resp.Msg.GetCarModel().GetName())
	}
}

func TestGetCarModelNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetCarModel(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarModelRequest{Id: 99999}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

func TestListCarModelVariantsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListCarModelVariants(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarModelVariantsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListCarModelVariantsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "All Variants Manufacturer")
	model := seedCarModelV2(t, repo, mfr.ID, "Variant Parent Model")
	variant1 := seedCarModelVariant(t, repo, model.ID, "Variant A")
	variant2 := seedCarModelVariant(t, repo, model.ID, "Variant B")

	resp, err := svc.ListCarModelVariants(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarModelVariantsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
	}

	if !ids[uint32(variant1.ID)] {
		t.Errorf("variant1 (id=%d) not found in response", variant1.ID)
	}
	if !ids[uint32(variant2.ID)] {
		t.Errorf("variant2 (id=%d) not found in response", variant2.ID)
	}
}

func TestListCarModelVariantsByModelID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "Filtered Variants Manufacturer")
	model1 := seedCarModelV2(t, repo, mfr.ID, "Filtered Variant Parent One")
	model2 := seedCarModelV2(t, repo, mfr.ID, "Filtered Variant Parent Two")
	variant1 := seedCarModelVariant(t, repo, model1.ID, "Variant One")
	_ = seedCarModelVariant(t, repo, model2.ID, "Variant Two")

	resp, err := svc.ListCarModelVariants(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarModelVariantsRequest{
			ModelId: uint32(model1.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetId() != uint32(variant1.ID) {
		t.Errorf("expected variant id %d, got %d", variant1.ID, items[0].GetId())
	}
	if items[0].GetModelId() != uint32(model1.ID) {
		t.Errorf("expected model id %d, got %d", model1.ID, items[0].GetModelId())
	}
}

func TestGetCarModelVariantSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "Success Variant Manufacturer")
	model := seedCarModelV2(t, repo, mfr.ID, "Success Variant Parent")
	variant := seedCarModelVariant(t, repo, model.ID, "Success Variant")

	resp, err := svc.GetCarModelVariant(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarModelVariantRequest{
			Id: uint32(variant.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetCarModelVariant().GetId() != uint32(variant.ID) {
		t.Errorf("expected id %d, got %d", variant.ID, resp.Msg.GetCarModelVariant().GetId())
	}
	if resp.Msg.GetCarModelVariant().GetModelId() != uint32(model.ID) {
		t.Errorf(
			"expected model id %d, got %d",
			model.ID,
			resp.Msg.GetCarModelVariant().GetModelId(),
		)
	}
	if resp.Msg.GetCarModelVariant().GetName() != "Success Variant" {
		t.Errorf(
			"expected name %q, got %q",
			"Success Variant",
			resp.Msg.GetCarModelVariant().GetName(),
		)
	}
}

func TestGetCarModelVariantNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetCarModelVariant(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarModelVariantRequest{Id: 99999}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}

//nolint:whitespace // multiline signature style
func seedCarClass(
	t *testing.T,
	repo rootrepo.Repository,
	name string,
) *models.CarClass {
	t.Helper()
	cc, err := repo.Cars().CarClasses().Create(context.Background(), &models.CarClassSetter{
		Name:      omit.From(name),
		IsActive:  omit.From(true),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car class %q: %v", name, err)
	}
	return cc
}

func TestListCarClassesEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListCarClasses(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarClassesRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListCarClassesReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	class1 := seedCarClass(t, repo, "GT3")
	class2 := seedCarClass(t, repo, "GT4")

	resp, err := svc.ListCarClasses(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarClassesRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	ids := make(map[uint32]bool)
	for _, item := range items {
		ids[item.GetId()] = true
	}

	if !ids[uint32(class1.ID)] {
		t.Errorf("class1 (id=%d) not found in response", class1.ID)
	}
	if !ids[uint32(class2.ID)] {
		t.Errorf("class2 (id=%d) not found in response", class2.ID)
	}
}

func TestGetCarClassSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	cc := seedCarClass(t, repo, "LMP2")

	resp, err := svc.GetCarClass(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarClassRequest{
			Id: uint32(cc.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetCarClass().GetId() != uint32(cc.ID) {
		t.Errorf("expected id %d, got %d", cc.ID, resp.Msg.GetCarClass().GetId())
	}
	if resp.Msg.GetCarClass().GetName() != "LMP2" {
		t.Errorf("expected name %q, got %q", "LMP2", resp.Msg.GetCarClass().GetName())
	}
}

func TestGetCarClassNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetCarClass(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarClassRequest{
			Id: 99999,
		}),
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected connect error, got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v", connectErr.Code())
	}
}
