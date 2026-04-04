package importer

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnsupportedSimulation indicates that no processor exists for a simulation.
	ErrUnsupportedSimulation = errors.New("unsupported simulation")

	// ErrUnsupportedFormat indicates that a processor does not support a format.
	ErrUnsupportedFormat = errors.New("unsupported import format")
	providers            = make(map[string]ProcessImport)
)

func Register(simulationName string, processor ProcessImport) {
	if processor == nil {
		return
	}
	simulationName = normalizeSimulationName(simulationName)
	providers[simulationName] = processor
}

// Factory resolves simulation-specific processors.
type Factory struct {
	processors map[string]ProcessImport
}

// NewFactory builds a factory from simulation-name registrations.
func NewFactory(registrations map[string]ProcessImport) *Factory {
	processors := make(map[string]ProcessImport, len(registrations))
	for name, processor := range registrations {
		if processor == nil {
			continue
		}
		processors[normalizeSimulationName(name)] = processor
	}

	return &Factory{processors: processors}
}

// NewDefaultFactory returns the default simulation processor registrations.
func NewDefaultFactory() *Factory {
	return NewFactory(providers)
}

// Get returns the processor for the given simulation name.
func (f *Factory) Get(simulationName string) (ProcessImport, error) {
	if f == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedSimulation, simulationName)
	}

	processor, ok := f.processors[normalizeSimulationName(simulationName)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedSimulation, simulationName)
	}

	return processor, nil
}

func normalizeSimulationName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
