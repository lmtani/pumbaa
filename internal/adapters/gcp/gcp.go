package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

type GCP struct {
	Aud     string
	Factory DependencyFactory
}

func NewGoogleCloud(aud string, factory DependencyFactory) *GCP {
	return &GCP{
		Aud:     aud,
		Factory: factory,
	}
}

func (gc *GCP) GetStorageClient(ctx context.Context) (CloudStorageClient, error) {
	client, err := gc.Factory.NewStorageClient(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Println("Credentials are available for Google Cloud Storage")
	return client, nil
}

func (gc *GCP) GetIAPToken(ctx context.Context) (string, error) {
	ts, err := gc.Factory.NewTokenSource(ctx, gc.Aud)
	if err != nil {
		return "", err
	}
	token, err := ts.Token()
	if err != nil {
		return "", err
	}
	return token.AccessToken, nil
}

// Wrapper wraps the dependencies used in this package
type Wrapper struct{}

func (r *Wrapper) NewStorageClient(ctx context.Context) (CloudStorageClient, error) {
	return storage.NewClient(ctx)
}

func (r *Wrapper) NewTokenSource(ctx context.Context, aud string) (oauth2.TokenSource, error) {
	return idtoken.NewTokenSource(ctx, aud)
}

// MockDependencyFactory implementations for testing
type MockDependencyFactory struct{}

func (m *MockDependencyFactory) NewStorageClient(ctx context.Context) (CloudStorageClient, error) {
	return &mockStorageClient{}, nil
}

func (m *MockDependencyFactory) NewTokenSource(ctx context.Context, aud string) (oauth2.TokenSource, error) {
	return &mockTokenSource{}, nil
}

type mockStorageClient struct{}

func (m *mockStorageClient) Close() error {
	return nil
}

type mockTokenSource struct{}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "fake-access-token"}, nil
}
