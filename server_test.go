package simplemockserver_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/daolis/simplemockserver/simplemockserver"
)

func TestNewMockServer(t *testing.T) {
	mockServer, err := NewMockServer(WithFile("testfiles/mock.json"), WithFixedPort(12345))
	require.NoError(t, err)
	defer mockServer.Stop()

	mockServer.GetURL()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	// wait for signal to stop
	<-signals
}
