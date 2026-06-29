package main

import (
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bitrise-io/go-utils/fileutil"
)

func Test_ensureFastlaneVersionAndCreateCmdSlice(t *testing.T) {
	gemfilePath := path.Join("testdata", "Gemfile")

	tests := []struct {
		name         string
		forceVersion string
		gemfilePth   string
		want         []string
		want1        string
		wantErr      bool
	}{
		{
			name:       "test bundler install",
			gemfilePth: gemfilePath,
			want:       []string{"bundle", "_2.4.12_", "exec", "fastlane"},
			want1:      "testdata",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ensureFastlaneVersionAndCreateCmdSlice(tt.forceVersion, tt.gemfilePth)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureFastlaneVersionAndCreateCmdSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ensureFastlaneVersionAndCreateCmdSlice() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ensureFastlaneVersionAndCreateCmdSlice() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_writeReleaseNotes(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "release_notes_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		releaseNotes string
		language     string
		wantPath     string
		wantContent  string
		wantErr      bool
	}{
		{
			name:         "empty release notes",
			releaseNotes: "",
			language:     "en-US",
			wantPath:     "",
			wantContent:  "",
			wantErr:      false,
		},
		{
			name:         "valid release notes with default language",
			releaseNotes: "Some amazing update notes!",
			language:     "",
			wantPath:     filepath.Join(tempDir, "metadata", "en-US", "release_notes.txt"),
			wantContent:  "Some amazing update notes!",
			wantErr:      false,
		},
		{
			name:         "valid release notes with custom language",
			releaseNotes: "Des notes de mise à jour!",
			language:     "fr-FR",
			wantPath:     filepath.Join(tempDir, "metadata", "fr-FR", "release_notes.txt"),
			wantContent:  "Des notes de mise à jour!",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, err := writeReleaseNotes(tempDir, tt.releaseNotes, tt.language)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeReleaseNotes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotPath != tt.wantPath {
				t.Errorf("writeReleaseNotes() gotPath = %v, want %v", gotPath, tt.wantPath)
			}

			if tt.wantPath != "" {
				content, err := fileutil.ReadStringFromFile(gotPath)
				if err != nil {
					t.Errorf("failed to read written release notes file: %s", err)
				}
				if content != tt.wantContent {
					t.Errorf("written file content = %v, want %v", content, tt.wantContent)
				}
			}
		})
	}
}
