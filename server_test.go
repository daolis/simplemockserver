package simplemockserver_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/daolis/simplemockserver"
)

func TestManualStartMockServer(t *testing.T) {
	manualTest, err := strconv.ParseBool(os.Getenv("MANUAL"))
	require.NoError(t, err)
	if !manualTest {
		t.Skip("Skipping manual test")
	}

	mockServer, err := NewMockServer(
		WithFile("testfiles/mock.yaml"),
		WithFixedPort(12345),
		WithCustomEndpoints(NewCustomEndpoints(CustomResponseEndpoint{
			"/custom": {Method: http.MethodGet, Responses: []CustomResponseEntry{
				{
					RequestQueryFn: nil,
					ResponseFn: func(w http.ResponseWriter, r *http.Request) error {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte("{\"custom\": \"blaa\"}"))
						return err
					},
				},
			},
			},
		})))
	require.NoError(t, err)
	defer mockServer.Stop()

	mockServer.GetURL()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	// wait for signal to stop
	<-signals
}

func TestEndpoints(t *testing.T) {
	mockServer, err := NewMockServer(
		WithFile("testfiles/mock.yaml"),
		WithCustomEndpoints(NewCustomEndpoints(CustomResponseEndpoint{
			"/custom": {Method: http.MethodGet, Responses: []CustomResponseEntry{
				{
					RequestQueryFn: func(r *http.Request) bool {
						return r.URL.Query().Get("name") == "test"
					},
					ResponseFn: func(w http.ResponseWriter, r *http.Request) error {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte("{\"custom\": \"valueForQuery\"}"))
						return err
					},
				},
				{
					RequestQueryFn: nil,
					ResponseFn: func(w http.ResponseWriter, r *http.Request) error {
						w.WriteHeader(http.StatusOK)
						_, err := w.Write([]byte("{\"custom\": \"blaa\"}"))
						return err
					},
				},
			},
			},
		})))
	require.NoError(t, err)
	defer mockServer.Stop()

	mockServerURL := mockServer.GetURL()

	client := http.DefaultClient

	type args struct {
		method   string
		endpoint string
		urlQuery string
		body     string
	}
	type jsonCheck struct {
		get   func(body map[string]any) string
		value string
	}
	type want struct {
		status     int
		jsonChecks []jsonCheck
	}
	tests := []struct {
		name    string
		args    args
		want    want
		wantErr bool
	}{
		{
			name: "endpoint1_withoutQuery",
			args: args{
				method:   "GET",
				endpoint: "/endpoint1",
				body:     "",
			},
			want: want{
				status: http.StatusNotFound,
				jsonChecks: []jsonCheck{
					{
						get: func(body map[string]any) string {
							return body["message"].(string)
						},
						value: "No response matches for endpoint '/endpoint1': Queries: URL[name = 'John']",
					},
				},
			},
		},
		{
			name: "endpoint1_witValidQuery",
			args: args{
				method:   "GET",
				endpoint: "/endpoint1?name=John",
				body:     "",
			},
			want: want{
				status: http.StatusNotFound,
				jsonChecks: []jsonCheck{
					{
						get: func(body map[string]any) string {
							return body["testKey1"].(string)
						},
						value: "testValue1",
					},
					{
						get: func(body map[string]any) string {
							return body["testKey2"].(string)
						},
						value: "testValue2",
					},
				},
			},
		},
		{
			name: "customWithMatchingQuery",
			args: args{
				method:   "GET",
				endpoint: "/custom?name=test",
				body:     "",
			},
			want: want{
				status: http.StatusOK,
				jsonChecks: []jsonCheck{
					{
						get: func(body map[string]any) string {
							return body["custom"].(string)
						},
						value: "valueForQuery",
					},
				},
			},
		},
		{
			name: "customWithNOTMatchingQuery",
			args: args{
				method:   "GET",
				endpoint: "/custom?name=invalid",
				body:     "",
			},
			want: want{
				status: http.StatusOK,
				jsonChecks: []jsonCheck{
					{
						get: func(body map[string]any) string {
							return body["custom"].(string)
						},
						value: "blaa",
					},
				},
			},
		},
		{
			name: "customWithoutQuery",
			args: args{
				method:   "GET",
				endpoint: "/custom",
				body:     "",
			},
			want: want{
				status: http.StatusOK,
				jsonChecks: []jsonCheck{
					{
						get: func(body map[string]any) string {
							return body["custom"].(string)
						},
						value: "blaa",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if len(tt.args.body) != 0 {
				bodyReader = strings.NewReader(tt.args.body)
			}
			request, err := http.NewRequest(tt.args.method, mockServerURL+tt.args.endpoint, bodyReader)
			require.NoError(t, err)
			request.Header.Add("Accept", "application/json")
			request.Header.Add("Content-Type", "application/json")
			resp, err := client.Do(request)
			require.NoError(t, err)

			require.Equal(t, tt.want.status, resp.StatusCode)
			fmt.Println(resp.Status)
			defer resp.Body.Close()
			var bodyData interface{}

			err = json.NewDecoder(resp.Body).Decode(&bodyData)
			require.NoError(t, err)
			for idx, check := range tt.want.jsonChecks {
				require.Equal(t, check.value, check.get(bodyData.(map[string]any)), "json check #%d failed", idx)
			}
		})
	}
}

func TestNewMockServer(t *testing.T) {
}
