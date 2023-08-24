package main

import (
	"path"
	"reflect"
	"testing"
)

func Test_ensureFastlaneVersionAndCreateCmdSlice(t *testing.T) {
	//gemfileLockPath := path.Join("testdata", "Gemfile.lock")
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
