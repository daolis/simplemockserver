package simplemockserver

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
)

const defaultMockFile = "testfiles/mock.json"

var mockFile MockFile
var customEndpoints CustomEndpoints

type MockServer interface {
	Stop()
	GetURL() string
}

type mockServer struct {
	port         int
	server       *httptest.Server
	mockFileName string
	fixedPort    int
	//customEndpoints CustomEndpoints
}

var _ MockServer = &mockServer{}

func (m mockServer) Stop() {
	m.server.Close()
}

func (m mockServer) GetURL() string {
	return m.server.URL
}

type MockServerProperty func(*mockServer)

func WithFile(file string) MockServerProperty {
	return func(m *mockServer) {
		m.mockFileName = file
	}
}

func WithFixedPort(port int) MockServerProperty {
	return func(m *mockServer) {
		m.fixedPort = port
	}
}

func WithCustomEndpoints(endpoints CustomEndpoints) MockServerProperty {
	return func(m *mockServer) {
		customEndpoints = endpoints
	}
}

func NewMockServer(properties ...MockServerProperty) (MockServer, error) {
	server := &mockServer{
		mockFileName: defaultMockFile,
		fixedPort:    0,
	}

	for _, property := range properties {
		property(server)
	}

	var err error
	mockFile, err = readMockFile(server.mockFileName)
	if err != nil {
		return nil, err
	}
	handlerFunc := http.HandlerFunc(jsonFileEndpointsHandler)
	if server.fixedPort != 0 {
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", server.fixedPort))
		if err != nil {
			return nil, err
		}
		server.server = &httptest.Server{
			Listener: l,
			Config:   &http.Server{Handler: handlerFunc},
		}
		server.server.Start()
	} else {
		server.server = httptest.NewServer(handlerFunc)
	}

	fmt.Printf("Started mock server at %s\n", server.server.URL)
	for key, endpoint := range customEndpoints {
		for method, _ := range endpoint {
			fmt.Printf(" [%6s] %s\n", method, path.Join(server.server.URL, key))
		}
	}
	for p, endpoint := range mockFile {
		for method, _ := range endpoint {
			var disabled string
			if customEndpoints != nil {
				if cep, ok := customEndpoints[p]; ok {
					if _, ok := cep[method]; ok {
						disabled = " - overruled by custom endpoint"
					}
				}
			}
			fmt.Printf(" [%6s] %s%s\n", method, path.Join(server.server.URL, p), disabled)
		}
	}
	return server, nil
}
