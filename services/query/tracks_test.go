//nolint:lll // test files can have some duplication and long lines for test data setup
package query

import (
	"context"
	"errors"
	"testing"

	queryv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/query/v1"
	"connectrpc.com/connect"
)

func TestListTracksEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListTracks(
		context.Background(),
		connect.NewRequest(&queryv1.ListTracksRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListTracksReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	monza := seedTrack(t, repo, "Monza")
	spa := seedTrack(t, repo, "Spa")

	resp, err := svc.ListTracks(
		context.Background(),
		connect.NewRequest(&queryv1.ListTracksRequest{}),
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

	if !ids[uint32(monza.ID)] {
		t.Errorf("monza track (id=%d) not found in response", monza.ID)
	}
	if !ids[uint32(spa.ID)] {
		t.Errorf("spa track (id=%d) not found in response", spa.ID)
	}
}

func TestGetTrackSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	track := seedTrack(t, repo, "Silverstone")

	resp, err := svc.GetTrack(
		context.Background(),
		connect.NewRequest(&queryv1.GetTrackRequest{
			Id: uint32(track.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetTrack().GetId() != uint32(track.ID) {
		t.Errorf("expected id %d, got %d", track.ID, resp.Msg.GetTrack().GetId())
	}
	if resp.Msg.GetTrack().GetName() != "Silverstone" {
		t.Errorf("expected name %q, got %q", "Silverstone", resp.Msg.GetTrack().GetName())
	}
}

func TestGetTrackNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetTrack(
		context.Background(),
		connect.NewRequest(&queryv1.GetTrackRequest{
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

func TestListTrackLayoutsEmpty(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	resp, err := svc.ListTrackLayouts(
		context.Background(),
		connect.NewRequest(&queryv1.ListTrackLayoutsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.GetItems()) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.Msg.GetItems()))
	}
}

func TestListTrackLayoutsReturnsAll(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	track := seedTrack(t, repo, "Monza")
	seedTrackLayout(t, repo, track.ID, "GP Circuit")
	seedTrackLayout(t, repo, track.ID, "Junior Circuit")

	resp, err := svc.ListTrackLayouts(
		context.Background(),
		connect.NewRequest(&queryv1.ListTrackLayoutsRequest{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListTrackLayoutsByTrackID(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	track1 := seedTrack(t, repo, "Monza")
	track2 := seedTrack(t, repo, "Spa")
	layout1 := seedTrackLayout(t, repo, track1.ID, "GP Circuit")
	seedTrackLayout(t, repo, track2.ID, "Full Circuit")

	resp, err := svc.ListTrackLayouts(
		context.Background(),
		connect.NewRequest(&queryv1.ListTrackLayoutsRequest{
			TrackId: uint32(track1.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := resp.Msg.GetItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].GetTrackId() != uint32(track1.ID) {
		t.Errorf("expected track_id %d, got %d", track1.ID, items[0].GetTrackId())
	}
	if items[0].GetId() != uint32(layout1.ID) {
		t.Errorf("expected layout id %d, got %d", layout1.ID, items[0].GetId())
	}
}

func TestGetTrackLayoutSuccess(t *testing.T) {
	svc, repo := newDBBackedQueryService(t)

	track := seedTrack(t, repo, "Spa")
	layout := seedTrackLayout(t, repo, track.ID, "Full Circuit")

	resp, err := svc.GetTrackLayout(
		context.Background(),
		connect.NewRequest(&queryv1.GetTrackLayoutRequest{
			Id: uint32(layout.ID),
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.GetTrackLayout().GetId() != uint32(layout.ID) {
		t.Errorf("expected id %d, got %d", layout.ID, resp.Msg.GetTrackLayout().GetId())
	}
	if resp.Msg.GetTrackLayout().GetTrackId() != uint32(track.ID) {
		t.Errorf("expected track_id %d, got %d", track.ID, resp.Msg.GetTrackLayout().GetTrackId())
	}
	if resp.Msg.GetTrackLayout().GetName() != "Full Circuit" {
		t.Errorf("expected name %q, got %q", "Full Circuit", resp.Msg.GetTrackLayout().GetName())
	}
}

func TestGetTrackLayoutNotFound(t *testing.T) {
	svc, _ := newDBBackedQueryService(t)

	_, err := svc.GetTrackLayout(
		context.Background(),
		connect.NewRequest(&queryv1.GetTrackLayoutRequest{
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
