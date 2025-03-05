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
	"errors"
	"testing"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/service/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sys/windows/svc"
)

func TestWindowsServiceHandler(t *testing.T) {
	t.Run("Normal start/stop", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}

		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(make(chan error)))
		rSvc.On("Stop", mock.Anything).Return(nil)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		changeChan <- svc.ChangeRequest{
			Cmd:           svc.Interrogate,
			CurrentStatus: svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown},
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for interrogate response")
		}

		changeChan <- svc.ChangeRequest{
			Cmd: svc.Stop,
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, uint32(0), statusCode, "status code was not 0")
	})

	t.Run("Start fails", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}

		startError := errors.New("Failed to start service")

		rSvc.On("Start", mock.Anything).Return(startError)
		rSvc.On("Error").Return((<-chan error)(make(chan error)))
		rSvc.On("Stop", mock.Anything).Return(nil)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, statusCodeServiceException, statusCode, "status code was not ServiceException")
	})

	t.Run("Unexpected service error", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}

		svcErr := errors.New("service unexpectedly failed")
		errChan := make(chan error, 1)
		errChan <- svcErr

		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(errChan))
		rSvc.On("Stop", mock.Anything).Return(nil)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, statusCodeServiceException, statusCode, "status code was not ServiceException")
	})

	t.Run("Shutdown error", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}
		stopError := errors.New("Failed to start service")

		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(make(chan error)))
		rSvc.On("Stop", mock.Anything).Return(stopError)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		changeChan <- svc.ChangeRequest{
			Cmd:           svc.Interrogate,
			CurrentStatus: svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown},
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for interrogate response")
		}

		changeChan <- svc.ChangeRequest{
			Cmd: svc.Stop,
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, statusCodeServiceException, statusCode, "status code was not ServiceException")
	})

	t.Run("Shutdown takes too long", func(t *testing.T) {
		setWindowsServiceTimeout(t, 10*time.Millisecond)
		rSvc := &mocks.MockRunnableService{}

		blockStopChan := make(chan struct{}, 1)
		t.Cleanup(func() {
			blockStopChan <- struct{}{}
		})

		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(make(chan error)))
		rSvc.On("Stop", mock.Anything).Run(func(args mock.Arguments) { <-blockStopChan }).Return(nil)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		changeChan <- svc.ChangeRequest{
			Cmd:           svc.Interrogate,
			CurrentStatus: svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown},
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for interrogate response")
		}

		changeChan <- svc.ChangeRequest{
			Cmd: svc.Stop,
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, statusCodeServiceException, statusCode, "status code was not ServiceException")
	})

	t.Run("Shutdown error after unexpected error", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}
		stopError := errors.New("Failed to start service")
		svcErr := errors.New("service unexpectedly failed")
		errChan := make(chan error, 1)
		errChan <- svcErr

		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(errChan))
		rSvc.On("Stop", mock.Anything).Return(stopError)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, statusCodeServiceException, statusCode, "status code was not ServiceException")
	})

	t.Run("Unhandled command", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}

		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(make(chan error)))
		rSvc.On("Stop", mock.Anything).Return(nil)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		changeChan <- svc.ChangeRequest{
			Cmd:           svc.Interrogate,
			CurrentStatus: svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown},
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for interrogate response")
		}

		changeChan <- svc.ChangeRequest{
			Cmd: svc.DeviceEvent,
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, uint32(statusCodeInvalidServiceCommand), statusCode, "status code was not InvalidServiceCommand")
	})

	t.Run("Unhandled command with shutdown error", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}

		stopError := errors.New("Failed to start service")
		rSvc.On("Start", mock.Anything).Return(nil)
		rSvc.On("Error").Return((<-chan error)(make(chan error)))
		rSvc.On("Stop", mock.Anything).Return(stopError)

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{"service-name"}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StartPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to start pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service status change to running")
		}

		changeChan <- svc.ChangeRequest{
			Cmd:           svc.Interrogate,
			CurrentStatus: svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown},
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for interrogate response")
		}

		changeChan <- svc.ChangeRequest{
			Cmd: svc.DeviceEvent,
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.StopPending}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stop pending")
		}

		select {
		case status := <-statusChan:
			require.Equal(t, svc.Status{State: svc.Stopped}, status)
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for status change to stopped")
		}

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, uint32(statusCodeServiceException), statusCode, "status code was not ServiceException")
	})

	t.Run("No service name", func(t *testing.T) {
		rSvc := &mocks.MockRunnableService{}

		svcHandler := newWindowsServiceHandler(zap.NewNop(), rSvc)

		changeChan := make(chan svc.ChangeRequest)
		statusChan := make(chan svc.Status, 6)
		svcHandlerDone := make(chan struct{})

		var isSvcSpecificStatus bool
		var statusCode uint32
		go func() {
			isSvcSpecificStatus, statusCode = svcHandler.Execute([]string{}, changeChan, statusChan)
			close(svcHandlerDone)
		}()

		select {
		case <-svcHandlerDone: // OK
		case <-time.After(time.Second):
			t.Fatalf("Timed out waiting for service handler to return")
		}

		require.Equal(t, false, isSvcSpecificStatus, "status code marked as service specific")
		require.Equal(t, uint32(statusCodeInvalidServiceName), statusCode, "status code was not InvalidServiceName")
	})
}

func setWindowsServiceTimeout(t *testing.T, d time.Duration) {
	old := windowsServiceShutdownTimeout
	windowsServiceShutdownTimeout = d
	t.Cleanup(func() {
		windowsServiceShutdownTimeout = old
	})
}
