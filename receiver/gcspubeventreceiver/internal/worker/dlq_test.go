// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Internal test file — uses package worker to access unexported symbols.
package worker

import (
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestDLQConditionKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		wantKind dlqErrorKind
	}{
		{
			name:     "nil error",
			err:      nil,
			wantKind: dlqErrorKindNone,
		},
		{
			name:     "storage.ErrObjectNotExist",
			err:      storage.ErrObjectNotExist,
			wantKind: dlqErrorKindFileNotFound,
		},
		{
			name:     "wrapped storage.ErrObjectNotExist",
			err:      fmt.Errorf("wrap: %w", storage.ErrObjectNotExist),
			wantKind: dlqErrorKindFileNotFound,
		},
		{
			name:     "googleapi 403 error",
			err:      &googleapi.Error{Code: 403},
			wantKind: dlqErrorKindPermissionDenied,
		},
		{
			name:     "wrapped googleapi 403 error",
			err:      fmt.Errorf("wrap: %w", &googleapi.Error{Code: 403}),
			wantKind: dlqErrorKindPermissionDenied,
		},
		{
			name:     "googleapi 404 error is not a DLQ condition",
			err:      &googleapi.Error{Code: 404},
			wantKind: dlqErrorKindNone,
		},
		{
			name:     "googleapi 500 error is not a DLQ condition",
			err:      &googleapi.Error{Code: 500},
			wantKind: dlqErrorKindNone,
		},
		{
			name:     "ErrNotArrayOrKnownObject",
			err:      ErrNotArrayOrKnownObject,
			wantKind: dlqErrorKindUnsupportedFile,
		},
		{
			name:     "generic network error",
			err:      errors.New("generic network error"),
			wantKind: dlqErrorKindNone,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := dlqConditionKind(tc.err)
			require.Equal(t, tc.wantKind, got)
		})
	}
}

func TestIsDLQConditionError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		err     error
		wantDLQ bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantDLQ: false,
		},
		{
			name:    "storage.ErrObjectNotExist",
			err:     storage.ErrObjectNotExist,
			wantDLQ: true,
		},
		{
			name:    "wrapped storage.ErrObjectNotExist",
			err:     fmt.Errorf("wrap: %w", storage.ErrObjectNotExist),
			wantDLQ: true,
		},
		{
			name:    "googleapi 403 error",
			err:     &googleapi.Error{Code: 403},
			wantDLQ: true,
		},
		{
			name:    "wrapped googleapi 403 error",
			err:     fmt.Errorf("wrap: %w", &googleapi.Error{Code: 403}),
			wantDLQ: true,
		},
		{
			name:    "googleapi 404 error is not a DLQ condition",
			err:     &googleapi.Error{Code: 404},
			wantDLQ: false,
		},
		{
			name:    "googleapi 500 error is not a DLQ condition",
			err:     &googleapi.Error{Code: 500},
			wantDLQ: false,
		},
		{
			name:    "ErrNotArrayOrKnownObject",
			err:     ErrNotArrayOrKnownObject,
			wantDLQ: true,
		},
		{
			name:    "generic network error",
			err:     errors.New("generic network error"),
			wantDLQ: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isDLQConditionError(tc.err)
			require.Equal(t, tc.wantDLQ, got)
		})
	}
}
