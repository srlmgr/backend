//go:build !unix

package tcpostgres

// withContainerLock is a no-op on non-unix platforms; local parallel test runs on
// those platforms may hit the same container-reuse race testcontainers-go has today.
func withContainerLock(fn func() error) error {
	return fn()
}
