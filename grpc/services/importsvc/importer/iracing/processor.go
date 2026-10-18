package iracing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/srlmgr/backend/grpc/services/conversion"
	processor "github.com/srlmgr/backend/grpc/services/importsvc/importer"
	"github.com/srlmgr/backend/log"
	"github.com/srlmgr/backend/support/iracing/irdata"
)

// Processor handles imports for iRacing simulations.
type Processor struct{}

var (
	_ processor.ProcessImport      = (*Processor)(nil)
	_ processor.MultiRaceProcessor = (*Processor)(nil)
)

func (p *Processor) SupportedFormats() []string {
	return []string{
		conversion.ImportFormatJSON,
		conversion.ImportFormatCSV,
		conversion.ImportFormatAPI,
	}
}

//nolint:whitespace // editor/linter issue
func (p *Processor) Process(
	ctx context.Context,
	format string,
	payload any,
) (*processor.ParsedImportPayload, error) {
	switch format {
	case conversion.ImportFormatJSON:
		parsed, err := ParseJSON(ctx, payload)
		if err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}

		return parsed, nil
	case conversion.ImportFormatCSV:
		parsed, err := ParseCSV(payload)
		if err != nil {
			return nil, fmt.Errorf("parse csv: %w", err)
		}

		return parsed, nil
	default:
		return nil, fmt.Errorf("%w: %s", processor.ErrUnsupportedFormat, format)
	}
}

// ProcessMultiRace parses payload and returns one ParsedImportPayload per race detected
// (heat races for JSON). For formats without heat-race support (CSV), it returns a
// single-element slice built from the standard Process result.
//
//nolint:whitespace // editor/linter issue
func (p *Processor) ProcessMultiRace(
	ctx context.Context,
	format string,
	payload any,
) ([]*processor.ParsedImportPayload, error) {
	switch format {
	case conversion.ImportFormatAPI:
		apiData, err := p.readFromAPI(ctx, payload)
		if err != nil {
			return nil, fmt.Errorf("read from api: %w", err)
		}
		parsed, err := ParseJSONMultiRace(ctx, apiData)
		if err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}
		return parsed, nil

	case conversion.ImportFormatJSON:
		parsed, err := ParseJSONMultiRace(ctx, payload)
		if err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}

		return parsed, nil
	default:
		parsed, err := p.Process(ctx, format, payload)
		if err != nil {
			return nil, err
		}

		return []*processor.ParsedImportPayload{parsed}, nil
	}
}

//nolint:whitespace // editor/linter issue
func (p *Processor) readFromAPI(
	ctx context.Context,
	payload any,
) (*EventResult, error) {
	api, ok := irdata.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing iRacing API context")
	}
	arg, ok := payload.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid payload type")
	}
	id, err := strconv.Atoi(string(arg))
	if err != nil {
		return nil, fmt.Errorf("invalid subsession ID: %w", err)
	}
	data, err := api.Get(
		strings.TrimSpace(fmt.Sprintf(`
			/data/results/get?subsession_id=%d
			`, id)),
	)
	if err != nil {
		log.Error("failed to get event result data", log.ErrorField(err))
		return nil, err
	}

	var eventResult EventResult
	err = json.Unmarshal(data, &eventResult)
	if err != nil {
		log.Error("failed to parse event result data", log.ErrorField(err))
		return nil, err
	}

	return &eventResult, nil
}

func init() {
	processor.Register("iracing", &Processor{})
}
