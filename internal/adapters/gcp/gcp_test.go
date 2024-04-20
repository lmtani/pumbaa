package gcp

import (
	"context"
	"reflect"
	"testing"
)

func TestGCP_GetIAPToken(t *testing.T) {
	type fields struct {
		Aud     string
		Factory DependencyFactory
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "GetIAPToken",
			fields: fields{
				Aud:     "fake-aud",
				Factory: &MockDependencyFactory{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    "fake-access-token",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &GCP{
				Aud:     tt.fields.Aud,
				Factory: tt.fields.Factory,
			}
			got, err := gc.GetIAPToken(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIAPToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetIAPToken() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGCP_GetStorageClient(t *testing.T) {
	type fields struct {
		Aud     string
		Factory DependencyFactory
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    CloudStorageClient
		wantErr bool
	}{
		{
			name: "GetStorageClient",
			fields: fields{
				Aud:     "fake-aud",
				Factory: &MockDependencyFactory{},
			},
			args: args{
				ctx: context.Background(),
			},
			want:    &mockStorageClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &GCP{
				Aud:     tt.fields.Aud,
				Factory: tt.fields.Factory,
			}
			got, err := gc.GetStorageClient(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStorageClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStorageClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGoogleCloud(t *testing.T) {
	type args struct {
		aud     string
		factory DependencyFactory
	}
	tests := []struct {
		name string
		args args
		want *GCP
	}{
		{
			name: "NewGoogleCloud",
			args: args{
				aud:     "fake-aud",
				factory: &MockDependencyFactory{},
			},
			want: &GCP{
				Aud:     "fake-aud",
				Factory: &MockDependencyFactory{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGoogleCloud(tt.args.aud, tt.args.factory); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGoogleCloud() = %v, want %v", got, tt.want)
			}
		})
	}
}
