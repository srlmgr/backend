package resolve

import (
	"github.com/srlmgr/backend/repository"
	"github.com/srlmgr/backend/services/importsvc/points"
)

type (
	Resolver struct {
		repos repository.Repository
	}
)

func NewResolver(repos repository.Repository) *Resolver {
	return &Resolver{
		repos: repos,
	}
}

func (r *Resolver) ResolverFunc() points.ResolveGridID {
	return func(gridID int32) (raceNo, gridNo int32, err error) {
		return 0, 0, nil
	}
}
