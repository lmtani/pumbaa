package gcp

import (
	"context"

	"golang.org/x/oauth2"
)

// CloudStorageClient defines the interface for storage client operations needed
type CloudStorageClient interface {
	Close() error
}

// DependencyFactory defines the interface for creating dependencies
type DependencyFactory interface {
	NewStorageClient(ctx context.Context) (CloudStorageClient, error)
	NewTokenSource(ctx context.Context, aud string) (oauth2.TokenSource, error)
}
