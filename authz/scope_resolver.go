package authz

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ResourceScope captures resolved scope identifiers used for authz policy checks.
type ResourceScope struct {
	SeriesID     string `json:"seriesId"`
	SimulationID string `json:"simulationId"`
	SeasonID     string `json:"seasonId"`
	EventID      string `json:"eventId"`
}

type scopeResolver struct {
	pool *pgxpool.Pool
}

func newScopeResolver(pool *pgxpool.Pool) *scopeResolver {
	return &scopeResolver{pool: pool}
}

//nolint:whitespace // multiline signature for line-length compliance
func (r *scopeResolver) Resolve(
	ctx context.Context,
	req connect.AnyRequest,
	policy ProcedurePolicy,
) (ResourceScope, error) {
	scope := ResourceScope{
		SeriesID:     strings.TrimSpace(req.Header().Get("x-series-id")),
		SimulationID: strings.TrimSpace(req.Header().Get("x-simulation-id")),
	}

	msg, ok := req.Any().(proto.Message)
	if ok {
		r.enrichFromMessage(msg, &scope)
	}

	if scope.EventID != "" {
		if err := r.resolveFromEvent(ctx, &scope); err != nil {
			return ResourceScope{}, err
		}
	}
	if scope.SeasonID != "" {
		if err := r.resolveFromSeason(ctx, &scope); err != nil {
			return ResourceScope{}, err
		}
	}
	if scope.SeriesID != "" {
		if err := r.resolveFromSeries(ctx, &scope); err != nil {
			return ResourceScope{}, err
		}
	}

	_ = policy
	return scope, nil
}

func (r *scopeResolver) enrichFromMessage(msg proto.Message, scope *ResourceScope) {
	if scope.EventID == "" {
		scope.EventID = extractIDField(msg, "event_id", "eventId", "event_frontend_id")
	}
	if scope.SeasonID == "" {
		scope.SeasonID = extractIDField(msg, "season_id", "seasonId", "season_frontend_id")
	}
	if scope.SeriesID == "" {
		scope.SeriesID = extractIDField(msg, "series_id", "seriesId", "series_frontend_id")
	}
	if scope.SimulationID == "" {
		scope.SimulationID = extractIDField(
			msg,
			"simulation_id",
			"simulationId",
			"simulation_frontend_id",
		)
	}
}

//nolint:whitespace // multiline signature for line-length compliance
func (r *scopeResolver) resolveFromEvent(
	ctx context.Context,
	scope *ResourceScope,
) error {
	const query = `
	SELECT s.id::text, sr.id::text, sr.simulation_id::text
	FROM events e
	JOIN seasons s ON s.id = e.season_id
	JOIN series sr ON sr.id = s.series_id
	WHERE e.id::text = $1 OR e.frontend_id::text = $1
	LIMIT 1`
	if err := r.pool.QueryRow(ctx, query, scope.EventID).Scan(
		&scope.SeasonID,
		&scope.SeriesID,
		&scope.SimulationID,
	); err != nil {
		return fmt.Errorf("resolve scope from event id: %w", err)
	}
	return nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (r *scopeResolver) resolveFromSeason(
	ctx context.Context,
	scope *ResourceScope,
) error {
	const query = `
	SELECT s.id::text, s.simulation_id::text
	FROM series s
	JOIN seasons se ON se.series_id = s.id
	WHERE se.id::text = $1 OR se.frontend_id::text = $1
	LIMIT 1`
	if err := r.pool.QueryRow(ctx, query, scope.SeasonID).Scan(
		&scope.SeriesID,
		&scope.SimulationID,
	); err != nil {
		return fmt.Errorf("resolve scope from season id: %w", err)
	}
	return nil
}

//nolint:whitespace // multiline signature for line-length compliance
func (r *scopeResolver) resolveFromSeries(
	ctx context.Context,
	scope *ResourceScope,
) error {
	const query = `
	SELECT simulation_id::text
	FROM series
	WHERE id::text = $1 OR frontend_id::text = $1
	LIMIT 1`
	if err := r.pool.QueryRow(ctx, query, scope.SeriesID).Scan(
		&scope.SimulationID,
	); err != nil {
		return fmt.Errorf("resolve scope from series id: %w", err)
	}
	return nil
}

func extractIDField(msg proto.Message, names ...string) string {
	ref := msg.ProtoReflect()
	for _, name := range names {
		fd := findFieldByName(ref, name)
		if fd == nil {
			continue
		}
		value := ref.Get(fd)
		//exhaustive:ignore We only handle scalar IDs that can map to scope values.
		switch fd.Kind() {
		case protoreflect.StringKind:
			if s := strings.TrimSpace(value.String()); s != "" {
				return s
			}
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			if value.Int() > 0 {
				return strconv.FormatInt(value.Int(), 10)
			}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			if value.Int() > 0 {
				return strconv.FormatInt(value.Int(), 10)
			}
		default:
			continue
		}
	}
	return ""
}

//nolint:whitespace // multiline signature for line-length compliance
func findFieldByName(
	msg protoreflect.Message,
	fieldName string,
) protoreflect.FieldDescriptor {
	descriptor := msg.Descriptor().Fields()
	for i := range descriptor.Len() {
		fd := descriptor.Get(i)
		if string(fd.Name()) == fieldName || fd.JSONName() == fieldName {
			return fd
		}
	}
	return nil
}
