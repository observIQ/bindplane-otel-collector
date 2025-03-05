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
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/service/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRunServiceInteractive(t *testing.T) {
	t.Run("Normal start/stop", func(t *testing.T) {
		svc := &mocks.MockRunnableService{}

		ctx, cancel := context.WithCancel(context.Background())

		svc.On("Start", mock.Anything).Return(nil)
		svc.On("Error").Return((<-chan error)(make(chan error)))
		svc.On("Stop", mock.Anything).Return(nil)

		var err error
		svcDone := make(chan struct{})
		go func() {
			err = runServiceInteractive(ctx, zap.NewNop(), svc)
			close(svcDone)
		}()

		<-time.After(500 * time.Millisecond)
		cancel()

		select {
		case <-svcDone: // OK
		case <-time.After(1 * time.Second):
			t.Fatalf("Timed out waiting for service done")
		}

		require.NoError(t, err)
	})

	t.Run("Start fails", func(t *testing.T) {
		svc := &mocks.MockRunnableService{}

		startErr := errors.New("failed to start")

		svc.On("Start", mock.Anything).Return(startErr)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var err error
		svcDone := make(chan struct{})
		go func() {
			err = runServiceInteractive(ctx, zap.NewNop(), svc)
			close(svcDone)
		}()

		select {
		case <-svcDone: // OK
		case <-time.After(1 * time.Second):
			t.Fatalf("Timed out waiting for service done")
		}

		require.Error(t, err)
		require.ErrorIs(t, err, startErr)
	})

	t.Run("Service errors", func(t *testing.T) {
		svc := &mocks.MockRunnableService{}

		svcErr := errors.New("service unexpectedly failed")
		errChan := make(chan error, 1)
		errChan <- svcErr

		svc.On("Start", mock.Anything).Return(nil)
		svc.On("Error").Return((<-chan error)(errChan))
		svc.On("Stop", mock.Anything).Return(nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var err error
		svcDone := make(chan struct{})
		go func() {
			err = runServiceInteractive(ctx, zap.NewNop(), svc)
			close(svcDone)
		}()

		select {
		case <-svcDone: // OK
		case <-time.After(1 * time.Second):
			t.Fatalf("Timed out waiting for service done")
		}

		require.Error(t, err)
		require.ErrorIs(t, err, svcErr)
	})

	t.Run("Stop errors", func(t *testing.T) {
		svc := &mocks.MockRunnableService{}

		stopErr := errors.New("Stop failed")

		svc.On("Start", mock.Anything).Return(nil)
		svc.On("Error").Return((<-chan error)(make(chan error)))
		svc.On("Stop", mock.Anything).Return(stopErr)

		ctx, cancel := context.WithCancel(context.Background())

		var err error
		svcDone := make(chan struct{})
		go func() {
			err = runServiceInteractive(ctx, zap.NewNop(), svc)
			close(svcDone)
		}()

		<-time.After(500 * time.Millisecond)
		cancel()

		select {
		case <-svcDone: // OK
		case <-time.After(1 * time.Second):
			t.Fatalf("Timed out waiting for service done")
		}

		require.Error(t, err)
		require.ErrorIs(t, err, stopErr)
	})

	t.Run("Stop errors after error returned", func(t *testing.T) {
		svc := &mocks.MockRunnableService{}

		stopErr := errors.New("Stop failed")
		svcErr := errors.New("service unexpectedly failed")
		errChan := make(chan error, 1)
		errChan <- svcErr

		svc.On("Start", mock.Anything).Return(nil)
		svc.On("Error").Return((<-chan error)(errChan))
		svc.On("Stop", mock.Anything).Return(stopErr)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var err error
		svcDone := make(chan struct{})
		go func() {
			err = runServiceInteractive(ctx, zap.NewNop(), svc)
			close(svcDone)
		}()

		select {
		case <-svcDone: // OK
		case <-time.After(1 * time.Second):
			t.Fatalf("Timed out waiting for service done")
		}

		require.Error(t, err)
		require.ErrorIs(t, err, stopErr)
	})
}
