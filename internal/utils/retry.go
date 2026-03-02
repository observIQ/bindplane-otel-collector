// Copyright  observIQ, Inc.
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

// Package utils provides utility functions across the collector.
package utils

import (
	"net/http"
	"strconv"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const retryAfterHeader = "Retry-After"

// ShouldRetryGRPC returns true if the error should be retried and the retry delay if present in the response.
// Spec Reference: https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#failures
func ShouldRetryGRPC(err error) (bool, time.Duration) {
	st := status.Convert(err)
	code := st.Code()

	var retryInfo *errdetails.RetryInfo
	for _, detail := range st.Details() {
		if ri, ok := detail.(*errdetails.RetryInfo); ok {
			retryInfo = ri
			break
		}
	}

	retryDelay := getGRPCRetryDelay(retryInfo)

	switch code {
	case codes.Canceled,
		codes.DeadlineExceeded,
		codes.Aborted,
		codes.OutOfRange,
		codes.Unavailable,
		codes.DataLoss:
		return true, retryDelay
	case codes.ResourceExhausted:
		// Per Spec: Retry ResourceExhausted errors only if RetryInfo is provided by the server.
		return retryInfo != nil, retryDelay
	default:
		return false, 0
	}
}

// getGRPCRetryDelay extracts retry delay from RetryInfo for GRPC error handling
// Spec Reference: https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#otlpgrpc-throttling
func getGRPCRetryDelay(retryInfo *errdetails.RetryInfo) time.Duration {
	if retryInfo == nil || retryInfo.RetryDelay == nil {
		return 0
	}

	if retryInfo.RetryDelay.Seconds > 0 || retryInfo.RetryDelay.Nanos > 0 {
		return time.Duration(retryInfo.RetryDelay.Seconds)*time.Second + time.Duration(retryInfo.RetryDelay.Nanos)*time.Nanosecond
	}

	return 0
}

// ShouldRetryHTTP returns true if the HTTP status code should be retried and the retry delay if present in the response headers.
func ShouldRetryHTTP(resp *http.Response) (bool, time.Duration) {
	if resp == nil {
		return false, 0
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true, getHTTPRetryDelay(resp.Header)
	default:
		return false, 0
	}
}

// getHTTPRetryDelay extracts retry delay from Retry-After header for HTTP error handling
// The value may be a number of seconds or an HTTP date.
// RFC Reference: https://datatracker.ietf.org/doc/html/rfc7231#section-7.1.3
func getHTTPRetryDelay(headers http.Header) time.Duration {
	retryAfter := headers.Get(retryAfterHeader)
	if retryAfter == "" {
		return 0
	}

	if seconds, err := strconv.ParseInt(retryAfter, 10, 64); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if retryTime, err := time.Parse(time.RFC1123, retryAfter); err == nil {
		delay := time.Until(retryTime)
		if delay > 0 {
			return delay
		}
	}

	return 0
}
