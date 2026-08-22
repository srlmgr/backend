package cache

type IDType int

const (
	IDSeason IDType = iota
	IDEvent
	IDRaceGrid
)

type CompositeKey[K comparable] struct {
	Type IDType
	ID   K
}

func NewCompositeKey[K comparable](idType IDType, id K) CompositeKey[K] {
	return CompositeKey[K]{Type: idType, ID: id}
}
