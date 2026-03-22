package iracing

import (
	"context"
	"fmt"

	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

// Processor handles imports for iRacing simulations.
type Processor struct{}

var _ processor.ProcessImport = (*Processor)(nil)

func (p *Processor) SupportedFormats() []string {
	return []string{
		// conversion.ImportFormatJSON,
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

func init() {
	processor.Register("iracing", &Processor{})
}
