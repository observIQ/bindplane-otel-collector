// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporterutils

import (
	"net/http"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestShouldRetryGRPC_NonRetriableCodes(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
	}{
		{name: "Unknown", code: codes.Unknown},
		{name: "InvalidArgument", code: codes.InvalidArgument},
		{name: "NotFound", code: codes.NotFound},
		{name: "AlreadyExists", code: codes.AlreadyExists},
		{name: "PermissionDenied", code: codes.PermissionDenied},
		{name: "Unauthenticated", code: codes.Unauthenticated},
		{name: "FailedPrecondition", code: codes.FailedPrecondition},
		{name: "Unimplemented", code: codes.Unimplemented},
		{name: "Internal", code: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := status.Error(tt.code, "test")
			retry, _ := ShouldRetryGRPC(err)
			if retry {
				t.Fatalf("expected retry to be false for non-retriable code; got true")
			}
		})
	}
}

func TestShouldRetryGRPC_RetriableCodes(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
	}{
		{name: "Canceled", code: codes.Canceled},
		{name: "DeadlineExceeded", code: codes.DeadlineExceeded},
		{name: "Aborted", code: codes.Aborted},
		{name: "OutOfRange", code: codes.OutOfRange},
		{name: "Unavailable", code: codes.Unavailable},
		{name: "DataLoss", code: codes.DataLoss},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := status.New(tt.code, "test")
			withDetails, err := st.WithDetails(&errdetails.RetryInfo{
				RetryDelay: durationpb.New(3 * time.Second),
			})
			if err != nil {
				t.Fatalf("failed to add retry info details: %v", err)
			}

			retry, delay := ShouldRetryGRPC(withDetails.Err())
			if !retry {
				t.Fatalf("expected retry to be true for code %v; got false", tt.code)
			}
			if delay != 3*time.Second {
				t.Fatalf("expected delay to be 3s; got %v", delay)
			}
		})
	}
}

func TestShouldRetryGRPC_ResourceExhaustedRequiresRetryInfo(t *testing.T) {
	errWithoutDetails := status.Error(codes.ResourceExhausted, "no details")

	retry, _ := ShouldRetryGRPC(errWithoutDetails)
	if retry {
		t.Fatalf("expected retry to be false when RetryInfo is not present")
	}

	st := status.New(codes.ResourceExhausted, "with details")
	withDetails, err := st.WithDetails(&errdetails.RetryInfo{
		RetryDelay: durationpb.New(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("failed to add retry info details: %v", err)
	}

	retry, delay := ShouldRetryGRPC(withDetails.Err())
	if !retry {
		t.Fatalf("expected retry to be true when RetryInfo is present")
	}
	if delay != 5*time.Second {
		t.Fatalf("expected delay to be 5s; got %v", delay)
	}
}

func TestGetGRPCRetryDelay(t *testing.T) {
	if delay := getGRPCRetryDelay(nil); delay != 0 {
		t.Fatalf("expected delay to be 0 for nil RetryInfo; got %v", delay)
	}

	delay := getGRPCRetryDelay(&errdetails.RetryInfo{
		RetryDelay: durationpb.New(2 * time.Second),
	})
	if delay != 2*time.Second {
		t.Fatalf("expected delay to be 2s; got %v", delay)
	}

	delay = getGRPCRetryDelay(&errdetails.RetryInfo{
		RetryDelay: &durationpb.Duration{},
	})
	if delay != 0 {
		t.Fatalf("expected delay to be 0 for zero RetryDelay; got %v", delay)
	}
}

func TestShouldRetryHTTP_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		// Retryable status codes
		{name: "TooManyRequests", statusCode: http.StatusTooManyRequests, expected: true},
		{name: "BadGateway", statusCode: http.StatusBadGateway, expected: true},
		{name: "ServiceUnavailable", statusCode: http.StatusServiceUnavailable, expected: true},
		{name: "GatewayTimeout", statusCode: http.StatusGatewayTimeout, expected: true},

		// Some non-retryable status codes
		{name: "BadRequest", statusCode: http.StatusBadRequest, expected: false},
		{name: "Unauthorized", statusCode: http.StatusUnauthorized, expected: false},
		{name: "Forbidden", statusCode: http.StatusForbidden, expected: false},
		{name: "NotFound", statusCode: http.StatusNotFound, expected: false},
		{name: "MethodNotAllowed", statusCode: http.StatusMethodNotAllowed, expected: false},
		{name: "Conflict", statusCode: http.StatusConflict, expected: false},
		{name: "Gone", statusCode: http.StatusGone, expected: false},
		{name: "LengthRequired", statusCode: http.StatusLengthRequired, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     http.Header{},
			}
			resp.Header.Set(retryAfterHeader, "10")

			retry, _ := ShouldRetryHTTP(resp)
			if retry != tt.expected {
				t.Fatalf("expected retry=%v for status %d; got %v", tt.expected, tt.statusCode, retry)
			}
		})
	}
}

func TestGetHTTPRetryDelay_SecondsHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set(retryAfterHeader, "60")

	delay := getHTTPRetryDelay(headers)
	if delay != 60*time.Second {
		t.Fatalf("expected delay to be 60s; got %v", delay)
	}
}

func TestGetHTTPRetryDelay_HTTPDateHeader(t *testing.T) {
	headers := http.Header{}
	retryTime := time.Now().Add(10 * time.Second).UTC()
	headers.Set(retryAfterHeader, retryTime.Format(time.RFC1123))

	delay := getHTTPRetryDelay(headers)
	if delay <= 0 {
		t.Fatalf("expected delay to be > 0 for future date; got %v", delay)
	}

	if delay > 20*time.Second {
		t.Fatalf("expected delay to be reasonably close to 10s; got %v", delay)
	}
}

func TestGetHTTPRetryDelay_InvalidOrPastHeader(t *testing.T) {
	headers := http.Header{}

	// No header
	if delay := getHTTPRetryDelay(headers); delay != 0 {
		t.Fatalf("expected delay to be 0 when header is missing; got %v", delay)
	}

	// Invalid value
	headers.Set(retryAfterHeader, "not-a-number")
	if delay := getHTTPRetryDelay(headers); delay != 0 {
		t.Fatalf("expected delay to be 0 for invalid header; got %v", delay)
	}

	// Past date
	headers.Set(retryAfterHeader, time.Now().Add(-10*time.Second).UTC().Format(time.RFC1123))
	if delay := getHTTPRetryDelay(headers); delay != 0 {
		t.Fatalf("expected delay to be 0 for past date; got %v", delay)
	}
}
