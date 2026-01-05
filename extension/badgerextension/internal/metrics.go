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

// Package internal contains the internal constants for the badger extension
package internal // import "github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal"

const (
	// ExtensionAttribute is the attribute for the extension
	ExtensionAttribute = "extension"
	// StorageTypeAttribute is the attribute for the type of storage client
	StorageTypeAttribute = "storage_type"
	// StorageTypeLSM is the attribute value for the LSM storage type
	StorageTypeLSM = "lsm"
	// StorageTypeValueLog is the attribute value for the value log storage type
	StorageTypeValueLog = "value_log"
	// OperationTypeAttribute is the attribute for the type of operation performed
	OperationTypeAttribute = "operation_type"
	// OperationTypeGet is the attribute value for the get operation type
	OperationTypeGet = "get"
	// OperationTypeSet is the attribute value for the set operation type
	OperationTypeSet = "set"
	// OperationTypeDelete is the attribute value for the delete operation type
	OperationTypeDelete = "delete"
)
