//go:build nopg

package main

import "context"

// openStore for the `nopg` build tag: in-memory only (local dev without Postgres).
func openStore(context.Context, string) (Store, error) { return NewMemStore(), nil }
