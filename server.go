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

type MockServer interface {
	Stop()
	GetURL() string
}

type mockServer struct {
	port         int
	server       *httptest.Server
	mockFileName string
	fixedPort    int
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
	for p, endpoint := range mockFile {
		fmt.Printf(" [%6s] %s\n", endpoint.Method, path.Join(server.server.URL, p))
	}
	return server, nil
}
