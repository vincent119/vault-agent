package domain_test

import (
	"testing"

	"vault-agent/internal/syncer/domain"
)

func TestParseSecretRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		annotations map[string]string
		wantNil     bool
		wantErr     bool
		wantBackend string
		wantPath    string
		wantKeys    []string
	}{
		{
			name:    "nil annotations returns nil",
			wantNil: true,
		},
		{
			name:        "empty annotations returns nil",
			annotations: map[string]string{},
			wantNil:     true,
		},
		{
			name: "path provided uses default vault backend",
			annotations: map[string]string{
				domain.AnnotationPath: "myapp/config",
			},
			wantBackend: "vault",
			wantPath:    "myapp/config",
		},
		{
			name: "custom aws backend",
			annotations: map[string]string{
				domain.AnnotationBackend: "aws",
				domain.AnnotationPath:   "prod/myapp",
			},
			wantBackend: "aws",
			wantPath:    "prod/myapp",
		},
		{
			name: "inject=false still parses if path present",
			annotations: map[string]string{
				domain.AnnotationInject: "false",
				domain.AnnotationPath:  "myapp/config",
			},
			wantBackend: "vault",
			wantPath:    "myapp/config",
		},
		{
			name: "keys parsed from JSON array",
			annotations: map[string]string{
				domain.AnnotationPath: "myapp/config",
				domain.AnnotationKeys: `["DB_HOST","DB_PASS"]`,
			},
			wantBackend: "vault",
			wantPath:    "myapp/config",
			wantKeys:    []string{"DB_HOST", "DB_PASS"},
		},
		{
			name: "invalid keys JSON returns error",
			annotations: map[string]string{
				domain.AnnotationPath: "myapp/config",
				domain.AnnotationKeys: `not-valid-json`,
			},
			wantErr: true,
		},
		{
			name: "no path and no inject returns nil",
			annotations: map[string]string{
				domain.AnnotationInject: "true",
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseSecretRef(tc.annotations)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseSecretRef() error = %v, wantErr = %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if tc.wantNil {
				if got != nil {
					t.Errorf("want nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("want non-nil SecretRef, got nil")
			}
			if got.Backend != tc.wantBackend {
				t.Errorf("Backend = %q, want %q", got.Backend, tc.wantBackend)
			}
			if got.Path != tc.wantPath {
				t.Errorf("Path = %q, want %q", got.Path, tc.wantPath)
			}
			if len(got.Keys) != len(tc.wantKeys) {
				t.Errorf("Keys = %v, want %v", got.Keys, tc.wantKeys)
				return
			}
			for i, k := range tc.wantKeys {
				if got.Keys[i] != k {
					t.Errorf("Keys[%d] = %q, want %q", i, got.Keys[i], k)
				}
			}
		})
	}
}
