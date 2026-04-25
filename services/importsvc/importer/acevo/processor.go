package acevo

import (
	"context"
	"fmt"

	"github.com/srlmgr/backend/services/conversion"
	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

// Processor handles imports for Assetto Corsa Competizione simulations.
type Processor struct{}

var _ processor.ProcessImport = (*Processor)(nil)

func (p *Processor) SupportedFormats() []string {
	return []string{conversion.ImportFormatJSON}
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
	default:
		return nil, fmt.Errorf("%w: %s", processor.ErrUnsupportedFormat, format)
	}
}

func init() {
	processor.Register("assetto corsa evolutione", &Processor{})
	processor.Register("acevo", &Processor{})
}
