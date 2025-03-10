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

// Package action provides actions used in install
package action

// RollbackableAction is an interface to represents an install action that may be rolled back.
//
//go:generate mockery --name RollbackableAction --filename mock_rollbackable_action.go --structname MockRollbackableAction
type RollbackableAction interface {
	Rollback() error
}
