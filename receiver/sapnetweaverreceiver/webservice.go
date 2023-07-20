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

package sapnetweaverreceiver // import "github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver"

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/hooklift/gowsdl/soap"

	"github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver/internal/models"
)

type webService interface {
	GetInstanceProperties() (*models.GetInstancePropertiesResponse, error)
	GetAlertTree() (*models.GetAlertTreeResponse, error)
	GetQueueStatistic() (*models.GetQueueStatisticResponse, error)
	GetSystemInstanceList() (*models.GetSystemInstanceListResponse, error)
	GetProcessList() (*models.GetProcessListResponse, error)
	EnqGetStatistic() (*models.EnqGetStatisticResponse, error)
	ABAPGetSystemWPTable() (*models.ABAPGetSystemWPTableResponse, error)
	FindFile(args ...string) ([]string, error)
	DpmonExecute(paths string) (string, error)
	CertExecute(command string) ([]string, error)
}

type netweaverWebService struct {
	client *soap.Client
}

func newWebService(client *soap.Client) webService {
	return &netweaverWebService{
		client: client,
	}
}

func (s *netweaverWebService) GetQueueStatistic() (*models.GetQueueStatisticResponse, error) {
	request := &models.GetQueueStatistic{}
	response := &models.GetQueueStatisticResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *netweaverWebService) EnqGetStatistic() (*models.EnqGetStatisticResponse, error) {
	request := &models.EnqGetStatistic{}
	response := &models.EnqGetStatisticResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
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

func (s *netweaverWebService) GetSystemInstanceList() (*models.GetSystemInstanceListResponse, error) {
	request := &models.GetSystemInstanceList{}
	response := &models.GetSystemInstanceListResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *netweaverWebService) GetProcessList() (*models.GetProcessListResponse, error) {
	request := &models.GetProcessList{}
	response := &models.GetProcessListResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *netweaverWebService) ABAPGetSystemWPTable() (*models.ABAPGetSystemWPTableResponse, error) {
	request := &models.ABAPGetSystemWPTable{}
	response := &models.ABAPGetSystemWPTableResponse{}
	err := s.client.Call("''", request, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *netweaverWebService) FindFile(args ...string) ([]string, error) {
	resp, err := exec.Command("/usr/bin/find", args...).Output()
	if err != nil {
		return []string{}, err
	}

	return processFile(string(resp)), nil
}

func (s *netweaverWebService) DpmonExecute(paths string) (string, error) {
	cmd := exec.Command("bash", "-c", paths)

	var output bytes.Buffer
	cmd.Stdout = &output

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return output.String(), nil
}

func (s *netweaverWebService) CertExecute(command string) ([]string, error) {
	cmd := exec.Command("bash", "-c", command)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		return []string{}, err
	}

	return processFile(output.String()), nil
}

func processFile(output string) []string {
	return strings.Split(string(output), "\n")
}
