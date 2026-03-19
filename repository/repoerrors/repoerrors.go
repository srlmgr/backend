// Package repoerrors defines shared sentinel errors for the repository layer.
package repoerrors

import "errors"

// ErrNotFound is returned when a requested entity does not exist in the store.
var ErrNotFound = errors.New("not found")
