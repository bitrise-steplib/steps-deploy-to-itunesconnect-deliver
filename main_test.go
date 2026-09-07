package main

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bitrise-io/go-steputils/v2/ruby"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	v2log "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/mocks"
	"github.com/stretchr/testify/mock"
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
	rubyFactory := newRubyCommandFactory(t)
	cmdFactory := command.NewFactory(env.NewRepository())

	tests := []struct {
		name       string
		invocation fastlaneInvocation
		want       string
	}{
		{
			name:       "system installed Fastlane",
			invocation: fastlaneInvocation{cmdFactory: cmdFactory},
			want:       `fastlane "deliver"`,
		},
		{
			// The gem lockfile does not name a bundler version, but Fastlane still has to be called
			// through bundler, otherwise the version the Gemfile pins is bypassed.
			name:       "bundler without a version",
			invocation: fastlaneInvocation{useBundler: true, rubyFactory: rubyFactory, cmdFactory: cmdFactory},
			want:       `bundle "exec" "fastlane" "deliver"`,
		},
		{
			name:       "bundler with a version",
			invocation: fastlaneInvocation{useBundler: true, bundlerVersion: "2.4.12", rubyFactory: rubyFactory, cmdFactory: cmdFactory},
			want:       `bundle "_2.4.12_" "exec" "fastlane" "deliver"`,
		},
		{
			name:       "Fastlane version selector",
			invocation: fastlaneInvocation{gemVersion: "2.217.0", rubyFactory: rubyFactory, cmdFactory: cmdFactory},
			want:       `fastlane "_2.217.0_" "deliver"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.invocation.createCommand([]string{"deliver"}, nil)

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
			got, got1, err := ensureFastlaneVersion(rubyFactory, false, command.NewFactory(env.NewRepository()), tt.forceVersion, tt.gemfilePth)
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

func Test_ensureFastlaneVersion_rubyMissing(t *testing.T) {
	// A nil Ruby command factory asserts that no branch reaches for Ruby when it is missing.
	var noRubyFactory ruby.CommandFactory

	tests := []struct {
		name         string
		forceVersion string
		gemfilePth   string
		wantErr      bool
	}{
		{
			name:         "a Fastlane version input needs Ruby",
			forceVersion: "2.0.0",
			wantErr:      true,
		},
		{
			name:       "a Gemfile needs Ruby",
			gemfilePth: path.Join("testdata", "Gemfile"),
			wantErr:    true,
		},
		{
			name:    "the system installed Fastlane does not need Ruby",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invocation, _, err := ensureFastlaneVersion(noRubyFactory, true, command.NewFactory(env.NewRepository()), tt.forceVersion, tt.gemfilePth)
			if tt.wantErr {
				if !errors.Is(err, ruby.ErrRubyNotFound) {
					t.Errorf("ensureFastlaneVersion() error = %v, want it to wrap ruby.ErrRubyNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ensureFastlaneVersion() error = %v, want nil", err)
			}
			if invocation.useBundler || invocation.gemVersion != "" {
				t.Errorf("ensureFastlaneVersion() = %+v, want the system installed Fastlane", invocation)
			}
		})
	}
}

// stubCommand is a command that reports success without running anything.
func stubCommand(t *testing.T) *mocks.Command {
	t.Helper()

	cmd := mocks.NewCommand(t)
	cmd.EXPECT().PrintableCommandArgs().Return("").Maybe()
	cmd.EXPECT().Run().Return(nil).Maybe()
	cmd.EXPECT().RunAndReturnTrimmedCombinedOutput().Return("", nil).Maybe()

	return cmd
}

// Test_ensureFastlaneVersion_builder covers which fastlaneInvocation the builder produces, and how
// it drives the Ruby command factory to get there. Unlike Test_ensureFastlaneVersion it installs
// nothing, so it can afford to cover every branch.
//
// It asserts the arguments the builder passes, not the command line they turn into: that rendering
// belongs to go-steputils and is covered by its own tests.
func Test_ensureFastlaneVersion_builder(t *testing.T) {
	const lockWithFastlaneAndBundler = `GEM
  remote: https://rubygems.org/
  specs:
    fastlane (2.214.0)

DEPENDENCIES
  fastlane (~> 2.214.0)

BUNDLED WITH
   2.4.12
`
	// A lockfile can pin Fastlane without naming a bundler version.
	const lockWithFastlaneOnly = `GEM
  remote: https://rubygems.org/
  specs:
    fastlane (2.214.0)

DEPENDENCIES
  fastlane (~> 2.214.0)
`
	const lockWithoutFastlane = `GEM
  remote: https://rubygems.org/
  specs:
    cocoapods (1.12.1)

DEPENDENCIES
  cocoapods (~> 1.12)

BUNDLED WITH
   2.4.12
`

	tests := []struct {
		name string
		// forceVersion is the Fastlane version input.
		forceVersion string
		// gemfile writes a Gemfile in the test's temp dir when true.
		gemfile bool
		// lockContent writes a gem lockfile next to the Gemfile when not empty.
		lockContent string
		// missingGemfile points the Gemfile path at a file that was never written.
		missingGemfile bool

		// expectRubyCommands declares the Ruby commands the branch has to create. It is nil for the
		// branches that must not touch Ruby at all: the mock fails the test on any call it was not
		// told to expect.
		expectRubyCommands func(t *testing.T, rubyFactory *mocks.RubyCommandFactory)

		wantUseBundler bool
		wantBundlerVer string
		wantGemVersion string
		// wantWorkDirIsGemfileDir expects the Gemfile's directory as the working directory.
		wantWorkDirIsGemfileDir bool
		wantErr                 bool
	}{
		{
			name: "no Fastlane version and no Gemfile uses the system installed Fastlane",
		},
		{
			name:           "a Gemfile path that does not exist uses the system installed Fastlane",
			gemfile:        true,
			missingGemfile: true,
		},
		{
			name:        "a lockfile pinning Fastlane and bundler calls Fastlane through that bundler",
			gemfile:     true,
			lockContent: lockWithFastlaneAndBundler,
			expectRubyCommands: func(t *testing.T, rubyFactory *mocks.RubyCommandFactory) {
				rubyFactory.EXPECT().CreateGemInstall("bundler", "2.4.12", false, true, mock.Anything).
					Return([]command.Command{stubCommand(t)}).Once()
				rubyFactory.EXPECT().CreateBundleInstall("2.4.12", mock.Anything).
					Return(stubCommand(t)).Once()
			},
			wantUseBundler:          true,
			wantBundlerVer:          "2.4.12",
			wantWorkDirIsGemfileDir: true,
		},
		{
			// The regression guarded here: useBundler must not be inferred from bundlerVersion, or
			// a Gemfile without a pinned bundler would bypass the Gemfile entirely.
			name:        "a lockfile pinning Fastlane but no bundler still calls Fastlane through bundler",
			gemfile:     true,
			lockContent: lockWithFastlaneOnly,
			expectRubyCommands: func(t *testing.T, rubyFactory *mocks.RubyCommandFactory) {
				rubyFactory.EXPECT().CreateGemInstall("bundler", "", false, true, mock.Anything).
					Return([]command.Command{stubCommand(t)}).Once()
				rubyFactory.EXPECT().CreateBundleInstall("", mock.Anything).
					Return(stubCommand(t)).Once()
			},
			wantUseBundler:          true,
			wantBundlerVer:          "",
			wantWorkDirIsGemfileDir: true,
		},
		{
			name:        "a lockfile that does not pin Fastlane uses the system installed Fastlane",
			gemfile:     true,
			lockContent: lockWithoutFastlane,
		},
		{
			// With no lockfile the builder runs `bundle install` to create one. Nothing is really
			// installed here, so the lockfile stays missing and that has to be reported.
			name:    "a Gemfile with no lockfile runs bundle install first",
			gemfile: true,
			expectRubyCommands: func(t *testing.T, rubyFactory *mocks.RubyCommandFactory) {
				rubyFactory.EXPECT().CreateBundleInstall("", mock.Anything).
					Return(stubCommand(t)).Once()
			},
			wantErr: true,
		},
		{
			name:         "a Fastlane version input installs that version and selects it",
			forceVersion: "2.217.0",
			expectRubyCommands: func(t *testing.T, rubyFactory *mocks.RubyCommandFactory) {
				rubyFactory.EXPECT().CreateGemInstall("fastlane", "2.217.0", false, false, mock.Anything).
					Return([]command.Command{stubCommand(t)}).Once()
			},
			wantGemVersion: "2.217.0",
		},
		{
			name:         "the latest stable Fastlane is installed without a version selector",
			forceVersion: latestStable,
			expectRubyCommands: func(t *testing.T, rubyFactory *mocks.RubyCommandFactory) {
				rubyFactory.EXPECT().CreateGemInstall("fastlane", "", false, false, mock.Anything).
					Return([]command.Command{stubCommand(t)}).Once()
			},
			wantGemVersion: "",
		},
		{
			name:         "the latest Fastlane prerelease is installed without a version selector",
			forceVersion: latestPrerelease,
			expectRubyCommands: func(t *testing.T, rubyFactory *mocks.RubyCommandFactory) {
				rubyFactory.EXPECT().CreateGemInstall("fastlane", "", true, false, mock.Anything).
					Return([]command.Command{stubCommand(t)}).Once()
			},
			wantGemVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gemfileDir := t.TempDir()
			gemfilePth := ""
			if tt.gemfile {
				gemfilePth = filepath.Join(gemfileDir, "Gemfile")
				if !tt.missingGemfile {
					if err := os.WriteFile(gemfilePth, []byte("source 'https://rubygems.org'\n\ngem 'fastlane'\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if tt.lockContent != "" {
				if err := os.WriteFile(filepath.Join(gemfileDir, "Gemfile.lock"), []byte(tt.lockContent), 0600); err != nil {
					t.Fatal(err)
				}
			}

			rubyFactory := mocks.NewRubyCommandFactory(t)
			if tt.expectRubyCommands != nil {
				tt.expectRubyCommands(t, rubyFactory)
			}

			invocation, workDir, err := ensureFastlaneVersion(rubyFactory, false, command.NewFactory(env.NewRepository()), tt.forceVersion, gemfilePth)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ensureFastlaneVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if invocation.useBundler != tt.wantUseBundler {
				t.Errorf("ensureFastlaneVersion() useBundler = %v, want %v", invocation.useBundler, tt.wantUseBundler)
			}
			if invocation.bundlerVersion != tt.wantBundlerVer {
				t.Errorf("ensureFastlaneVersion() bundlerVersion = %q, want %q", invocation.bundlerVersion, tt.wantBundlerVer)
			}
			if invocation.gemVersion != tt.wantGemVersion {
				t.Errorf("ensureFastlaneVersion() gemVersion = %q, want %q", invocation.gemVersion, tt.wantGemVersion)
			}
			wantWorkDir := ""
			if tt.wantWorkDirIsGemfileDir {
				wantWorkDir = gemfileDir
			}
			if workDir != wantWorkDir {
				t.Errorf("ensureFastlaneVersion() workDir = %q, want %q", workDir, wantWorkDir)
			}
		})
	}
}
