package acc

import (
	"context"
	"fmt"

	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

// Processor handles imports for Assetto Corsa Competizione simulations.
type Processor struct{}

var _ processor.ProcessImport = (*Processor)(nil)

func (p *Processor) SupportedFormats() []string {
	return []string{conversion.ImportFormatJSON, conversion.ImportFormatCSV}
}

//nolint:whitespace // editor/linter issue
func (p *Processor) Process(
	_ context.Context,
	_ string,
	_ any,
) (*processor.ParsedImportPayload, error) {
	return nil, fmt.Errorf("assetto corsa competizione processor is not implemented")
}

func init() {
	processor.Register("assetto corsa competizione", &Processor{})
	processor.Register("acc", &Processor{})
}
