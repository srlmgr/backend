//nolint:lll // test files can have some duplication and long lines for test data setup
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

func seedCarManufacturer(t *testing.T, repo rootrepo.Repository, name string) *models.CarManufacturer {
	t.Helper()
	m, err := repo.Cars().CarManufacturers().Create(context.Background(), &models.CarManufacturerSetter{
		Name:      omit.From(name),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car manufacturer %q: %v", name, err)
	}
	return m
}

func seedCarBrand(t *testing.T, repo rootrepo.Repository, manufacturerID int32, name string) *models.CarBrand {
	t.Helper()
	b, err := repo.Cars().CarBrands().Create(context.Background(), &models.CarBrandSetter{
		ManufacturerID: omit.From(manufacturerID),
		Name:           omit.From(name),
		CreatedBy:      omit.From(testUserSeed),
		UpdatedBy:      omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car brand %q: %v", name, err)
	}
	return b
}

func seedCarModel(t *testing.T, repo rootrepo.Repository, brandID int32, name string) *models.CarModel {
	t.Helper()
	cm, err := repo.Cars().CarModels().Create(context.Background(), &models.CarModelSetter{
		BrandID:   omit.From(brandID),
		Name:      omit.From(name),
		CreatedBy: omit.From(testUserSeed),
		UpdatedBy: omit.From(testUserSeed),
	})
	if err != nil {
		t.Fatalf("failed to seed car model %q: %v", name, err)
	}
	return cm
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
		t.Errorf("expected name %q, got %q", "Apex Motorsports", resp.Msg.GetCarManufacturer().GetName())
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

func TestListCarBrandsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListCarBrands(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarBrandsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListCarBrandsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "Global Cars")
	alpha := seedCarBrand(t, repo, mfr.ID, "Alpha Brand")
	beta := seedCarBrand(t, repo, mfr.ID, "Beta Brand")

	resp, err := svc.ListCarBrands(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarBrandsRequest{}),
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
		t.Errorf("alpha brand (id=%d) not found in response", alpha.ID)
	}
	if !ids[uint32(beta.ID)] {
		t.Errorf("beta brand (id=%d) not found in response", beta.ID)
	}
}

func TestListCarBrandsByManufacturerID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr1 := seedCarManufacturer(t, repo, "First Manufacturer")
	mfr2 := seedCarManufacturer(t, repo, "Second Manufacturer")
	brand1 := seedCarBrand(t, repo, mfr1.ID, "Brand One")
	_ = seedCarBrand(t, repo, mfr2.ID, "Brand Two")

	resp, err := svc.ListCarBrands(
		context.Background(),
		connect.NewRequest(&queryv1.ListCarBrandsRequest{
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
	if items[0].GetId() != uint32(brand1.ID) {
		t.Errorf("expected brand id %d, got %d", brand1.ID, items[0].GetId())
	}
	if items[0].GetManufacturerId() != uint32(mfr1.ID) {
		t.Errorf("expected manufacturer id %d, got %d", mfr1.ID, items[0].GetManufacturerId())
	}
}

func TestGetCarBrandSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "Test Manufacturer")
	brand := seedCarBrand(t, repo, mfr.ID, "Test Brand")

	resp, err := svc.GetCarBrand(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarBrandRequest{
			Id: uint32(brand.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetCarBrand().GetId() != uint32(brand.ID) {
		t.Errorf("expected id %d, got %d", brand.ID, resp.Msg.GetCarBrand().GetId())
	}
	if resp.Msg.GetCarBrand().GetManufacturerId() != uint32(mfr.ID) {
		t.Errorf("expected manufacturer id %d, got %d", mfr.ID, resp.Msg.GetCarBrand().GetManufacturerId())
	}
	if resp.Msg.GetCarBrand().GetName() != "Test Brand" {
		t.Errorf("expected name %q, got %q", "Test Brand", resp.Msg.GetCarBrand().GetName())
	}
}

func TestGetCarBrandNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetCarBrand(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarBrandRequest{
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

	mfr := seedCarManufacturer(t, repo, "All Models Manufacturer")
	brand := seedCarBrand(t, repo, mfr.ID, "All Models Brand")
	model1 := seedCarModel(t, repo, brand.ID, "Model X")
	model2 := seedCarModel(t, repo, brand.ID, "Model Y")

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

	mfr1 := seedCarManufacturer(t, repo, "Manufacturer One")
	mfr2 := seedCarManufacturer(t, repo, "Manufacturer Two")
	brand1 := seedCarBrand(t, repo, mfr1.ID, "Brand for Mfr One")
	brand2 := seedCarBrand(t, repo, mfr2.ID, "Brand for Mfr Two")
	model1 := seedCarModel(t, repo, brand1.ID, "Model for Mfr One")
	_ = seedCarModel(t, repo, brand2.ID, "Model for Mfr Two")

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
}

func TestGetCarModelSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	mfr := seedCarManufacturer(t, repo, "Success Manufacturer")
	brand := seedCarBrand(t, repo, mfr.ID, "Success Brand")
	cm := seedCarModel(t, repo, brand.ID, "Success Model")

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
	if resp.Msg.GetCarModel().GetBrandId() != uint32(brand.ID) {
		t.Errorf("expected brand id %d, got %d", brand.ID, resp.Msg.GetCarModel().GetBrandId())
	}
	if resp.Msg.GetCarModel().GetName() != "Success Model" {
		t.Errorf("expected name %q, got %q", "Success Model", resp.Msg.GetCarModel().GetName())
	}
}

func TestGetCarModelNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetCarModel(
		context.Background(),
		connect.NewRequest(&queryv1.GetCarModelRequest{
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
