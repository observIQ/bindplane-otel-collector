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

// Package models contain request and response structures used for the SAPControl Web Service Interface
package models // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/models"

import "encoding/xml"

// StateColor holds the string state color value
type StateColor string

// StateColorCode holds the numerical state color value
type StateColorCode int

// StateColor constant values
const (
	StateColorGray       StateColor     = "SAPControl-GRAY"
	StateColorGreen      StateColor     = "SAPControl-GREEN"
	StateColorYellow     StateColor     = "SAPControl-YELLOW"
	StateColorRed        StateColor     = "SAPControl-RED"
	StateColorCodeGray   StateColorCode = 1
	StateColorCodeGreen  StateColorCode = 2
	StateColorCodeYellow StateColorCode = 3
	StateColorCodeRed    StateColorCode = 4
)

// EnqGetLockTableResponse is an xml response struct
type EnqGetLockTableResponse struct {
	XMLName xml.Name   `xml:"urn:SAPControl EnqGetLockTableResponse"`
	EnqLock []*EnqLock `xml:"lock>item,omitempty" json:"lock>item,omitempty"`
}

// EnqLock is an xml response struct
type EnqLock struct {
	LockName string `xml:"lock_name,omitempty" json:"lock_name,omitempty"`
	Object   string `xml:"object,omitempty" json:"object,omitempty"`
}

// GetAlertTreeResponse is an xml response struct
type GetAlertTreeResponse struct {
	XMLName   xml.Name     `xml:"urn:SAPControl GetAlertTreeResponse"`
	AlertNode []*AlertNode `xml:"tree>item,omitempty" json:"tree>item,omitempty"`
}

// AlertNode is an xml response struct
type AlertNode struct {
	Name        string     `xml:"name,omitempty" json:"name,omitempty"`
	ActualValue StateColor `xml:"ActualValue,omitempty" json:"ActualValue,omitempty"`
	Description string     `xml:"description,omitempty" json:"description,omitempty"`
}

// GetInstancePropertiesResponse is an xml response struct
type GetInstancePropertiesResponse struct {
	XMLName    xml.Name            `xml:"urn:SAPControl GetInstancePropertiesResponse"`
	Properties []*InstanceProperty `xml:"properties>item,omitempty" json:"properties>item,omitempty"`
}

// InstanceProperty is an xml response struct
type InstanceProperty struct {
	Property string `xml:"property,omitempty" json:"property,omitempty"`
	Value    string `xml:"value,omitempty" json:"value,omitempty"`
}

// CurrentSapInstance is a struct to hold the instance xml instant property response values
type CurrentSapInstance struct {
	Name     string
	Hostname string
}
