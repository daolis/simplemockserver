package simplemockserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

type Response struct {
	StatusCode int `json:"status" yaml:"status"`
	Body       any `json:"body" yaml:"body"`
}

type RequestQuery struct {
	Body *string `json:"body" yaml:"body"`
	URL  *string `json:"url" yaml:"url"`
}

type RequestQueryFn func(r *http.Request) bool
type ResponseFn func(w http.ResponseWriter, r *http.Request) error

type ResponseEntry struct {
	RequestQuery   *RequestQuery  `json:"requestQuery" yaml:"requestQuery"`
	RequestQueryFn RequestQueryFn `json:"-" yaml:"-"`
	Response       Response       `json:"response" yaml:"response"`
	ResponseFn     ResponseFn     `json:"-" yaml:"-"`
}

func (r ResponseEntry) hasQuery() bool {
	return (r.RequestQuery != nil && (r.RequestQuery.URL != nil || r.RequestQuery.Body != nil)) || r.RequestQueryFn != nil
}

type CustomResponseEntry struct {
	RequestQueryFn RequestQueryFn `json:"-"`
	ResponseFn     ResponseFn     `json:"-"`
}

type CustomResponseEndpoint map[string]struct {
	Method    string
	Responses []CustomResponseEntry
}

type Endpoint map[string][]ResponseEntry

type CustomEndpoints map[string]Endpoint

func NewCustomEndpoints(endpoints CustomResponseEndpoint) CustomEndpoints {
	eps := make(map[string]Endpoint, len(endpoints))
	for epPath, endpoint := range endpoints {
		responseEntries := make([]ResponseEntry, len(endpoint.Responses))
		for idx, resp := range endpoint.Responses {
			responseEntries[idx] = ResponseEntry{RequestQueryFn: resp.RequestQueryFn, ResponseFn: resp.ResponseFn}
		}
		eps[epPath] = Endpoint{
			endpoint.Method: responseEntries,
		}
	}
	return eps
}

type MockFile map[string]Endpoint

func readMockFile(filename string) (MockFile, error) {
	mockFile := make(MockFile)
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return mockFile, err
	}
	fileExt := strings.ToLower(path.Ext(filename))
	switch fileExt {
	case ".json":
		err = json.Unmarshal(file, &mockFile)
	case ".yaml":
		err = yaml.Unmarshal(file, &mockFile)
	default:
		return nil, fmt.Errorf("unsupported file type '%s'", fileExt)
	}
	if err != nil {
		return mockFile, err
	}
	return mockFile, nil
}
