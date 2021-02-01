package main

import (
	"reflect"
	"testing"

	"github.com/bitrise-steplib/steps-deploy-to-itunesconnect-deliver/appleauth"
)

func TestAppendFastlaneCredentials(t *testing.T) {
	type args struct {
		inParams   FastlaneParams
		authConfig appleauth.Credentials
	}
	tests := []struct {
		name    string
		args    args
		want    FastlaneParams
		wantErr bool
	}{
		{
			args: args{
				inParams: FastlaneParams{},
				authConfig: appleauth.Credentials{
					AppleID: &appleauth.AppleID{
						Username: "a@test.org",
					},
				},
			},
			want: FastlaneParams{
				Args: []string{"--username", "a@test.org"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AppendFastlaneCredentials(tt.args.inParams, tt.args.authConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("AppendFastlaneCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendFastlaneCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}
