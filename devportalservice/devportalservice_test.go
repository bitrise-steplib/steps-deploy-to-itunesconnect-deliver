package devportalservice

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAppleDeveloperConnection(t *testing.T) {
	tests := []struct {
		name string

		response *http.Response
		err      error

		want    *AppleDeveloperConnection
		wantErr bool
	}{
		{
			name: "No Apple Developer Connection set for the build",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{}")),
			},
			want:    &AppleDeveloperConnection{},
			wantErr: false,
		},
		{
			name: "No Apple Developer Connection set for the build, test devices available",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testDevicesResponseBody)),
			},
			want:    &testConnectionOnlyDevices,
			wantErr: false,
		},
		{
			name: "Session-based Apple Developer Connection set for the build",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testSessionConnectionResponseBody)),
			},
			want:    &testConnectionWithSessionConnection,
			wantErr: false,
		},
		{
			name: "JWT Apple Developer Connection set for the build",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testJWTConnectionResponseBody)),
			},
			want:    &testConnectionWithJWTConnection,
			wantErr: false,
		},
		{
			name: "Session-based and JWT Apple Developer Connection set for the build, test device available",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testSessionAndJWTConnectionResponseBody)),
			},
			want:    &testConnectionWithSessionAndJWTConnection,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBitriseClient(newMockHTTPClient(tt.response, nil), "dummy url", "dummy token")
			got, err := c.GetAppleDeveloperConnection()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSessionEnvValue(t *testing.T) {
	tests := []struct {
		name string

		response *http.Response
		err      error

		want    string
		wantErr bool
	}{
		{
			name: "No Apple Developer Connection set for the build",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader("{}")),
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Session-based Apple Developer Connection set for the build",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testSessionConnectionResponseBody)),
			},
			want:    testFastlaneSession,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBitriseClient(newMockHTTPClient(tt.response, nil), "dummy url", "dummy token")
			conn, err := c.GetAppleDeveloperConnection()
			require.NoError(t, err)

			if tt.want == "" {
				require.Nil(t, conn.SessionConnection)
				return
			}

			got, err := conn.SessionConnection.FastlaneLoginSession()
			if (err != nil) != tt.wantErr {
				t.Errorf("SessionData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SessionData() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockHTTPClient struct {
	response *http.Response
	err      error
}

func newMockHTTPClient(response *http.Response, err error) mockHTTPClient {
	return mockHTTPClient{response: response}
}

func (c mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return c.response, c.err
}

func restorableSetEnv(t *testing.T, key, value string) func() {
	origValue, set := os.LookupEnv(key)
	require.NoError(t, os.Setenv(key, value))
	if set {
		return func() { require.NoError(t, os.Setenv(key, origValue)) }
	}
	return func() { require.NoError(t, os.Unsetenv(key)) }

}
