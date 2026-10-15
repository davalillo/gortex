package main

import (
	"testing"

	"github.com/zzet/gortex/internal/config"
)

func TestResolveDaemonBackendPrecedence(t *testing.T) {
	tests := []struct {
		name string
		flag string
		env  string
		gc   *config.GlobalConfig
		want string
	}{
		{
			name: "flag wins over env and config",
			flag: "sqlite",
			env:  "memory",
			gc:   &config.GlobalConfig{Daemon: config.DaemonConfig{Backend: "mem"}},
			want: "sqlite",
		},
		{
			name: "env wins over config",
			flag: "",
			env:  "memory",
			gc:   &config.GlobalConfig{Daemon: config.DaemonConfig{Backend: "mem"}},
			want: "memory",
		},
		{
			name: "config applies when flag and env are empty",
			flag: "",
			env:  "",
			gc:   &config.GlobalConfig{Daemon: config.DaemonConfig{Backend: "memory"}},
			want: "memory",
		},
		{
			name: "nil config falls through to the sqlite default",
			flag: "",
			env:  "",
			gc:   nil,
			want: "sqlite",
		},
		{
			name: "empty config backend falls through to the sqlite default",
			flag: "",
			env:  "",
			gc:   &config.GlobalConfig{},
			want: "sqlite",
		},
		{
			name: "empty env value is ignored, config applies",
			flag: "",
			env:  "   ",
			gc:   &config.GlobalConfig{Daemon: config.DaemonConfig{Backend: "memory"}},
			want: "memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveDaemonBackend(tt.flag, tt.env, tt.gc)
			if got != tt.want {
				t.Fatalf("resolveDaemonBackend(%q, %q, %+v) = %q, want %q",
					tt.flag, tt.env, tt.gc, got, tt.want)
			}
		})
	}
}
