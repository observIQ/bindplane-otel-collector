package orphandetectorextension

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type parentWatcher struct {
	interval   time.Duration
	logger     *zap.Logger
	ppid       int
	ticker     *time.Ticker
	tickerDone chan bool
	tickCb     func()
	dieIfInit  bool
}

func newParentWatcher(interval time.Duration, dieIfInit bool, ppid int, logger *zap.Logger) *parentWatcher {
	return &parentWatcher{
		interval:  interval,
		logger:    logger,
		ppid:      ppid,
		dieIfInit: dieIfInit,
	}
}

func (pw *parentWatcher) Start(ctx context.Context, host component.Host) error {
	pw.ticker = time.NewTicker(pw.interval)
	pw.tickerDone = make(chan bool)
	go func() {
		for {
			select {
			case <-pw.tickerDone:
				return
			case <-pw.ticker.C:
				pw.onTick(host, pw.tickCb)
			}
		}
	}()

	pw.logger.Debug("Started parentWatcher", zap.Int("ppid", pw.ppid))

	return nil
}

func (pw *parentWatcher) onTick(host component.Host, cb func()) {
	if orphan(pw.ppid, pw.dieIfInit, pw.logger) {
		host.ReportFatalError(errors.New("process became an orphan"))
	} else if cb != nil {
		cb()
	}
}

func (pw *parentWatcher) Shutdown(ctx context.Context) error {
	pw.ticker.Stop()
	pw.tickerDone <- true

	return nil
}
