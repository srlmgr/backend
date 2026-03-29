package importer

import (
	"context"
	"errors"
	"testing"

	"github.com/srlmgr/backend/services/conversion"
)

type testProcessor struct {
	formats []ImportFormat
}

func (p *testProcessor) SupportedFormats() []ImportFormat {
	return p.formats
}

//nolint:whitespace // editor/linter issue
func (p *testProcessor) Process(
	_ context.Context,
	_ ImportFormat,
	_ any,
) (*ParsedImportPayload, error) {
	return nil, nil
}

func TestFactoryGetNormalizesSimulationName(t *testing.T) {
	t.Parallel()

	p := &testProcessor{formats: []ImportFormat{conversion.ImportFormatCSV}}
	factory := NewFactory(map[string]ProcessImport{
		"iRacing": p,
	})

	got, err := factory.Get("  IRACING ")
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if got != p {
		t.Fatal("Get returned unexpected processor instance")
	}
}

func TestFactoryGetUnsupportedSimulation(t *testing.T) {
	t.Parallel()

	factory := NewFactory(map[string]ProcessImport{})

	_, err := factory.Get("unknown-sim")
	if !errors.Is(err, ErrUnsupportedSimulation) {
		t.Fatalf("expected unsupported simulation error, got: %v", err)
	}
}

func TestSupportsFormat(t *testing.T) {
	t.Parallel()

	p := &testProcessor{formats: []ImportFormat{conversion.ImportFormatJSON}}

	if !SupportsFormat(p, conversion.ImportFormatJSON) {
		t.Fatal("expected json format to be supported")
	}
	if SupportsFormat(p, conversion.ImportFormatCSV) {
		t.Fatal("expected csv format to be unsupported")
	}
}
