package iracing

import (
	"context"
	"fmt"

	"github.com/srlmgr/backend/grpc/services/conversion"
	processor "github.com/srlmgr/backend/grpc/services/importsvc/importer"
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
		parsed, err := ParseJSON(payload)
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
	case conversion.ImportFormatJSON:
		parsed, err := ParseJSONMultiRace(payload)
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

func init() {
	processor.Register("iracing", &Processor{})
}
