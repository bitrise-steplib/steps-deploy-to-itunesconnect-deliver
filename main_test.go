package main

import (
	"path"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	v2log "github.com/bitrise-io/go-utils/v2/log"
)

func newRubyCommandFactory(t *testing.T) ruby.CommandFactory {
	t.Helper()

	logger := v2log.NewLogger()
	cmdFactory := command.NewFactory(env.NewRepository())
	rubyFactory, err := ruby.NewCommandFactory(cmdFactory, env.NewCommandLocator(), logger)
	if err != nil {
		t.Fatalf("failed to create Ruby command factory: %s", err)
	}

	return rubyFactory
}

func Test_fastlaneInvocation_createCommand(t *testing.T) {
	tests := []struct {
		name       string
		invocation fastlaneInvocation
		want       string
	}{
		{
			name:       "system installed Fastlane",
			invocation: fastlaneInvocation{},
			want:       `fastlane "deliver"`,
		},
		{
			// The gem lockfile does not name a bundler version, but Fastlane still has to be called
			// through bundler, otherwise the version the Gemfile pins is bypassed.
			name:       "bundler without a version",
			invocation: fastlaneInvocation{useBundler: true},
			want:       `bundle "exec" "fastlane" "deliver"`,
		},
		{
			name:       "bundler with a version",
			invocation: fastlaneInvocation{useBundler: true, bundlerVersion: "2.4.12"},
			want:       `bundle "_2.4.12_" "exec" "fastlane" "deliver"`,
		},
		{
			name:       "Fastlane version selector",
			invocation: fastlaneInvocation{gemVersion: "2.217.0"},
			want:       `fastlane "_2.217.0_" "deliver"`,
		},
	}

	rubyFactory := newRubyCommandFactory(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.invocation.createCommand(rubyFactory, []string{"deliver"}, nil)

			if got := cmd.PrintableCommandArgs(); got != tt.want {
				t.Errorf("createCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ensureFastlaneVersion(t *testing.T) {
	gemfilePath := path.Join("testdata", "Gemfile")

	tests := []struct {
		name               string
		forceVersion       string
		gemfilePth         string
		wantUseBundler     bool
		wantBundlerVersion string
		wantGemVersion     string
		wantWorkDir        string
		wantErr            bool
	}{
		{
			name:               "test bundler install",
			gemfilePth:         gemfilePath,
			wantUseBundler:     true,
			wantBundlerVersion: "2.4.12",
			wantWorkDir:        "testdata",
			wantErr:            false,
		},
		{
			name:           "no Gemfile and no version uses the system installed Fastlane",
			wantUseBundler: false,
			wantWorkDir:    "",
			wantErr:        false,
		},
	}

	rubyFactory := newRubyCommandFactory(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ensureFastlaneVersion(rubyFactory, tt.forceVersion, tt.gemfilePth)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureFastlaneVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.useBundler != tt.wantUseBundler {
				t.Errorf("ensureFastlaneVersion() useBundler = %v, want %v", got.useBundler, tt.wantUseBundler)
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
