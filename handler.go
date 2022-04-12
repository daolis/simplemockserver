package simplemockserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	if err != nil {
		w.WriteHeader(err.StatusCode)
		err := json.NewEncoder(w).Encode(ErrorResponse{Message: err.Message})
		if err != nil {
			return
		}
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) *MockServerError {
	if mockEndpoint, ok := mockFile[r.URL.Path]; ok {
		if mockEndpoint.Method != r.Method {
			return NewMockServerError(http.StatusNotFound, "Method '%s' not found for endpoint '%s'", r.Method, r.URL.Path)
		}

		var usedResponse *Response
		defer r.Body.Close()
		payload, err := io.ReadAll(r.Body)
		if err != nil {
			return NewMockServerError(http.StatusInternalServerError, "Could not read body for endpoint '%s'", r.URL.Path)
		}
		for idx, responseEntry := range mockEndpoint.Responses {
			if responseEntry.RequestQuery != nil {
				if responseEntry.RequestQuery.URL != nil {
					// convert map to json
					urlQuery, err := json.Marshal(r.URL.Query())
					if err != nil {
						return NewMockServerError(http.StatusInternalServerError, "Could not marshal query for endpoint '%s'", r.URL.Path)
					}
					parser, err := jsonql.NewStringQuery(string(urlQuery))
					query, err := parser.Query(*responseEntry.RequestQuery.URL)
					if err != nil {
						return NewMockServerError(http.StatusInternalServerError, "Could not parse URL query for endpoint '%s'", r.URL.Path)
					}
					if query != nil {
						fmt.Printf("Using response #%d for endpoint '%s'\n", idx, r.URL.Path)
						usedResponse = &responseEntry.Response
						break
					}
				}
				if (r.Method == http.MethodPost || r.Method == http.MethodPut) && responseEntry.RequestQuery.Body != nil {
					parser, err := jsonql.NewStringQuery(string(payload))
					query, err := parser.Query(*responseEntry.RequestQuery.Body)
					if err != nil {
						return NewMockServerError(http.StatusInternalServerError, "Could not parse body query for endpoint '%s'", r.URL.Path)
					}
					if query != nil {
						fmt.Printf("Using response #%d for endpoint '%s'\n", idx, r.URL.Path)
						usedResponse = &responseEntry.Response
						break
					}
				}
			}
		}
		if usedResponse == nil {
			return NewMockServerError(http.StatusNotFound, "Endpoint '%s' not found", r.URL.Path)
		}
		jsonString, err := json.Marshal(usedResponse.Body)
		if err != nil {
			return NewMockServerError(http.StatusInternalServerError, "Could not marshal response for endpoint '%s'", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(usedResponse.StatusCode)
		_, err = w.Write(jsonString)
		if err != nil {
			return NewMockServerError(http.StatusInternalServerError, "Could not write response for endpoint '%s'", r.URL.Path)
		}
	}
	return nil
}
