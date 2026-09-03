package main

import (
	"path"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/log"
)

func Test_ensureFastlaneVersion(t *testing.T) {
	gemfilePath := path.Join("testdata", "Gemfile")

	tests := []struct {
		name               string
		forceVersion       string
		gemfilePth         string
		wantBundlerVersion string
		wantGemVersion     string
		wantWorkDir        string
		wantErr            bool
	}{
		{
			name:               "test bundler install",
			gemfilePth:         gemfilePath,
			wantBundlerVersion: "2.4.12",
			wantWorkDir:        "testdata",
			wantErr:            false,
		},
	}

	logger := log.NewLogger()
	envRepository := env.NewRepository()
	cmdFactory := command.NewFactory(envRepository)
	rubyFactory, err := ruby.NewCommandFactory(cmdFactory, env.NewCommandLocator(), logger)
	if err != nil {
		t.Fatalf("failed to create Ruby command factory: %s", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ensureFastlaneVersion(rubyFactory, cmdFactory, tt.forceVersion, tt.gemfilePth)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureFastlaneVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.bundlerVersion != tt.wantBundlerVersion {
				t.Errorf("ensureFastlaneVersion() bundlerVersion = %v, want %v", got.bundlerVersion, tt.wantBundlerVersion)
			}
			if got.gemVersion != tt.wantGemVersion {
				t.Errorf("ensureFastlaneVersion() gemVersion = %v, want %v", got.gemVersion, tt.wantGemVersion)
			}
			if got1 != tt.wantWorkDir {
				t.Errorf("ensureFastlaneVersion() workDir = %v, want %v", got1, tt.wantWorkDir)
			}
		})
	}
}
