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

package sapnetweaverreceiver // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver"

import (
	"github.com/hooklift/gowsdl/soap"

	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/models"
)

type webService interface {
	GetInstanceProperties() (*models.GetInstancePropertiesResponse, error)
	GetAlertTree() (*models.GetAlertTreeResponse, error)
	EnqGetLockTable() (*models.EnqGetLockTableResponse, error)
}

type netweaverWebService struct {
	client *soap.Client
}

func newWebService(client *soap.Client) webService {
	return &netweaverWebService{
		client: client,
	}
}

func (s *netweaverWebService) GetAlertTree() (*models.GetAlertTreeResponse, error) {
	request := &models.GetAlertTree{}
	response := &models.GetAlertTreeResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *netweaverWebService) GetInstanceProperties() (*models.GetInstancePropertiesResponse, error) {
	request := &models.GetInstanceProperties{}
	response := &models.GetInstancePropertiesResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *netweaverWebService) EnqGetLockTable() (*models.EnqGetLockTableResponse, error) {
	request := &models.EnqGetLockTable{}
	response := &models.EnqGetLockTableResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}
