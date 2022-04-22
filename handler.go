package simplemockserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/elgs/jsonql"
)

type MockServerError struct {
	StatusCode int
	Message    string
}

func (m MockServerError) Error() string {
	return fmt.Sprintf("%d: %s", m.StatusCode, m.Message)
}

func NewMockServerError(statuscode int, format string, a ...interface{}) *MockServerError {
	return &MockServerError{
		StatusCode: statuscode,
		Message:    fmt.Sprintf(format, a...),
	}
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func jsonFileEndpointsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fmt.Println(r.URL)

	err := handleRequest(w, r)
	var mockServerErr *MockServerError
	if errors.As(err, &mockServerErr) {
		w.WriteHeader(mockServerErr.StatusCode)
		err := json.NewEncoder(w).Encode(ErrorResponse{Message: mockServerErr.Message})
		if err != nil {
			return
		}
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) error {
	responseEntry, err := findEndpointResponse(r)
	if err != nil {
		return err
	}
	// if ResponseFn is not nil, then it's a custom endpoint
	if responseEntry.ResponseFn != nil {
		return responseEntry.ResponseFn(w, r)
	}

	response := responseEntry.Response
	jsonString, err2 := json.Marshal(response.Body)
	if err2 != nil {
		return NewMockServerError(http.StatusInternalServerError, "Could not marshal response for endpoint '%s'", r.URL.Path)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_, err2 = w.Write(jsonString)
	if err2 != nil {
		return NewMockServerError(http.StatusInternalServerError, "Could not write response for endpoint '%s'", r.URL.Path)
	}
	return nil
}

func findEndpointResponse(r *http.Request) (*ResponseEntry, error) {
	getEndpointResponseEntries := func(r *http.Request) ([]ResponseEntry, error) {
		if cep, ok := customEndpoints[r.URL.Path]; ok {
			if cepm, ok := cep[r.Method]; ok {
				return cepm, nil
			}
		} else if mockEndpoint, ok := mockFile[r.URL.Path]; ok {
			if responses, ok := mockEndpoint[r.Method]; ok {
				return responses, nil
			}
		}
		return nil, NewMockServerError(http.StatusNotFound, "Endpoint '%s:%s' not found", r.URL.Path, r.Method)
	}

	responses, rErr := getEndpointResponseEntries(r)
	if rErr != nil {
		return nil, rErr
	}
	return getCorrectResponse(r, responses)
}

func getCorrectResponse(r *http.Request, responses []ResponseEntry) (*ResponseEntry, error) {
	var usedResponse *ResponseEntry
	var queries []string

	defer r.Body.Close()
	bodyPayload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, NewMockServerError(http.StatusInternalServerError, "Could not read body for endpoint '%s'", r.URL.Path)
	}

	for idx, responseEntry := range responses {
		if responseEntry.hasQuery() {
			if responseEntry.RequestQueryFn != nil {
				if responseEntry.RequestQueryFn(r) {
					return &responseEntry, nil
				}
				queries = append(queries, fmt.Sprintf("FunctionalFilter"))
				continue
			}
			if responseEntry.RequestQuery.URL != nil {
				// convert map to json string
				urlPayload, err := json.Marshal(r.URL.Query())
				if err != nil {
					return nil, NewMockServerError(http.StatusInternalServerError, "Could not marshal query for endpoint '%s'", r.URL.Path)
				}
				parser, err := jsonql.NewStringQuery(string(urlPayload))
				query, err := parser.Query(*responseEntry.RequestQuery.URL)
				if err != nil {
					return nil, NewMockServerError(http.StatusInternalServerError, "Could not parse URL query for endpoint '%s'", r.URL.Path)
				}
				if query != nil {
					fmt.Printf("Using response #%d for endpoint '%s'\n", idx, r.URL.Path)
					return &responseEntry, nil
				}
				queries = append(queries, fmt.Sprintf("URL[%s]", *responseEntry.RequestQuery.URL))
			}
			if (r.Method == http.MethodPost || r.Method == http.MethodPut) && responseEntry.RequestQuery.Body != nil {
				parser, err := jsonql.NewStringQuery(string(bodyPayload))
				query, err := parser.Query(*responseEntry.RequestQuery.Body)
				if err != nil {
					return nil, NewMockServerError(http.StatusInternalServerError, "Could not parse body query for endpoint '%s'", r.URL.Path)
				}
				if query != nil {
					fmt.Printf("Using response #%d for endpoint '%s'\n", idx, r.URL.Path)
					return &responseEntry, nil
				}
				queries = append(queries, fmt.Sprintf("URL[%s]", *responseEntry.RequestQuery.Body))
			}
		} else {
			usedResponse = &responseEntry
		}
	}
	if usedResponse == nil {
		return nil, NewMockServerError(http.StatusNotFound, "No response matches for endpoint '%s': Queries: %s", r.URL.Path, strings.Join(queries, ", "))
	}
	return usedResponse, nil
}
