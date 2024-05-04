package ports

import (
	"context"

	"github.com/lmtani/pumbaa/internal/adapters/gcp"
)

type GoogleCloudPlatform interface {
	GetStorageClient(ctx context.Context) (gcp.CloudStorageClient, error)
	GetIAPToken(ctx context.Context) (string, error)
}
