package pluginruntime

import "testing"

func TestValidateRepositoryURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "https", raw: "https://github.com/acme/plugin.git"},
		{name: "ssh scheme", raw: "ssh://git@github.com/acme/plugin.git"},
		{name: "scp style ssh", raw: "git@github.com:acme/plugin.git"},
		{name: "empty", raw: "", wantErr: true},
		{name: "file scheme", raw: "file:///tmp/plugin", wantErr: true},
		{name: "local path", raw: "../plugin", wantErr: true},
		{name: "http", raw: "http://github.com/acme/plugin.git", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepositoryURL(tt.raw)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
