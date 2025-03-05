// Copyright  Bindplane, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/observiq/bindplane-otel-collector/collector"
)

// StandaloneCollectorService is a RunnableService that runs the collector in standalone mode.
type StandaloneCollectorService struct {
	col      collector.Collector
	doneChan chan struct{}
	errChan  chan error
	wg       *sync.WaitGroup
}

// NewStandaloneCollectorService creates a new StandaloneCollectorService
func NewStandaloneCollectorService(c collector.Collector) StandaloneCollectorService {
	return StandaloneCollectorService{
		col:      c,
		doneChan: make(chan struct{}, 1),
		errChan:  make(chan error, 1),
		wg:       &sync.WaitGroup{},
	}
}

// Start starts the collector
func (s StandaloneCollectorService) Start(ctx context.Context) error {
	err := s.col.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed while starting collector: %w", err)
	}

	// monitor status for errors, so we don't zombie the service
	s.wg.Add(1)
	go s.monitorStatus()
	return nil
}

// monitorStatus monitors the collector's status for errors, and reports them
// to the error channel to trigger a shutdown.
func (s StandaloneCollectorService) monitorStatus() {
	defer s.wg.Done()
	statusChan := s.col.Status()
	for {
		select {
		case status := <-statusChan:
			// This will catch panics and errors
			if status.Err != nil {
				s.errChan <- status.Err
			} else if !status.Running {
				// If we aren't running, bail out. Otherwise the collector is effectively a "zombie" process.
				s.errChan <- errors.New("collector unexpectedly stopped running")
			}
		case <-s.doneChan:
			return
		}
	}
}

// Error returns a channel that can emit asynchronous, unrecoverable errors
func (s StandaloneCollectorService) Error() <-chan error {
	return s.errChan
}

// Stop shuts down the underlying collector
func (s StandaloneCollectorService) Stop(ctx context.Context) error {
	close(s.doneChan)

	collectorStoppedChan := make(chan struct{})
	go func() {
		s.col.Stop(ctx)
		s.wg.Wait()
		close(collectorStoppedChan)
	}()

	select {
	case <-collectorStoppedChan:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed while waiting for service shutdown: %w", ctx.Err())
	}
}
