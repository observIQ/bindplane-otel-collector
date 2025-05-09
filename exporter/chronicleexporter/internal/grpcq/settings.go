// Copyright observIQ, Inc.
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

package grpcq

import (
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func Settings() exporterhelper.QueueBatchSettings {
	return exporterhelper.QueueBatchSettings{
		Encoding: &Encoding{},
		/*
		 TODO this sizer pretends to measure 'requests' but actually measures 'bytes'
		 This is a workaround because upstream currently enforces that sizer must be
		  set to 'requests'. Once this restriction is removed, we should change the key
		  from RequestSizerTypeRequests to RequestSizerTypeBytes.
		*/
		Sizers: map[exporterhelper.RequestSizerType]exporterhelper.RequestSizer{
			exporterhelper.RequestSizerTypeRequests: &ByteSizer{},
		},
	}
}
