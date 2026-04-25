package lmu

import (
	"context"
	"fmt"

	"github.com/srlmgr/backend/services/conversion"
	processor "github.com/srlmgr/backend/services/importsvc/importer"
)

// Processor handles imports for Le Mans Ultimate simulations.
type Processor struct{}

var _ processor.ProcessImport = (*Processor)(nil)

func (p *Processor) SupportedFormats() []string {
	return []string{conversion.ImportFormatXML}
}

//nolint:whitespace // editor/linter issue
func (p *Processor) Process(
	ctx context.Context,
	format string,
	payload any,
) (*processor.ParsedImportPayload, error) {
	switch format {
	case conversion.ImportFormatXML:
		parsed, err := ParseXML(payload)
		if err != nil {
			return nil, fmt.Errorf("parse xml: %w", err)
		}

		return parsed, nil
	default:
		return nil, fmt.Errorf("%w: %s", processor.ErrUnsupportedFormat, format)
	}
}

func init() {
	processor.Register("lmu", &Processor{})
}
