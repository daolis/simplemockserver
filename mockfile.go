package simplemockserver

import (
	"encoding/json"
	"io/ioutil"
)

type Response struct {
	StatusCode int `json:"status"`
	Body       any `json:"body"`
}

type RequestQuery struct {
	Body *string `json:"body"`
	URL  *string `json:"url"`
}

type ResponseEntry struct {
	RequestQuery *RequestQuery `json:"requestQuery"`
	Response     Response      `json:"response"`
}

type Endpoint struct {
	Method    string          `json:"method"`
	Responses []ResponseEntry `json:"responses"`
}

type MockFile map[string]Endpoint

func readMockFile(filename string) (MockFile, error) {
	mockFile := make(MockFile)
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return mockFile, err
	}
	err = json.Unmarshal(file, &mockFile)
	if err != nil {
		return mockFile, err
	}
	return mockFile, nil
}
