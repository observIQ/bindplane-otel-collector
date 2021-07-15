package orphandetectorextension

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.uber.org/zap"
)

type testHost struct {
	reportFatalErrorCb func(err error)
}

func (th testHost) ReportFatalError(err error) {
	th.reportFatalErrorCb(err)
}

func (th testHost) GetFactory(_ component.Kind, _ config.Type) component.Factory {
	return nil
}

func (th testHost) GetExtensions() map[config.ComponentID]component.Extension {
	return nil
}

func (th testHost) GetExporters() map[config.DataType]map[config.ComponentID]component.Exporter {
	return nil
}

func TestParentWatcher(t *testing.T) {
	pw := newParentWatcher(500*time.Millisecond, false, os.Getppid(), zap.NewNop())
	require.NotNil(t, pw)

	triggeredChan := make(chan bool)

	pw.tickCb = func() {
		triggeredChan <- true
	}

	h := testHost{
		reportFatalErrorCb: func(err error) {
			require.Fail(t, "parent watcher detected that ppid changed while testing, but it shouldn't")
		},
	}

	err := pw.Start(context.Background(), h)
	require.NoError(t, err)

	timeout := time.NewTimer(10 * time.Second)
	select {
	case <-timeout.C:
		require.Fail(t, "Timed out before parent watcher ticked")
	case <-triggeredChan:
		timeout.Stop()
	}

	err = pw.Shutdown(context.Background())
	require.NoError(t, err)
}
