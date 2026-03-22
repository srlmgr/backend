package iracing

import (
	"context"
	"fmt"

	commonv1 "buf.build/gen/go/srlmgr/api/protocolbuffers/go/backend/common/v1"

	"github.com/srlmgr/backend/services/conversion"
	"github.com/srlmgr/backend/services/importsvc/processor"
)

// Processor handles imports for iRacing simulations.
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
) ([]*commonv1.ResultEntry, []*commonv1.UnresolvedMapping, error) {
	return nil, nil, fmt.Errorf("iracing processor is not implemented")
}

func init() {
	processor.Register("iracing", &Processor{})
}
