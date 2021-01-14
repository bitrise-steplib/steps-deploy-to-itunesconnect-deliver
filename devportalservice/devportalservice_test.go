package devportalservice

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
			name: "No Apple Developer Connection set for the build, test devices available",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testDevicesResponseBody)),
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Session-based Apple Developer Connection set for the build",
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(strings.NewReader(testAppleDevConnDataJSON)),
			},
			want:    testAppleDevConnSession,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewBitriseClient(newMockHTTPClient(tt.response, nil))
			conn, err := c.GetAppleDeveloperConnection("dummy url", "dummy token")
			require.NoError(t, err)

			got, err := conn.FastlaneLoginSession()
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
