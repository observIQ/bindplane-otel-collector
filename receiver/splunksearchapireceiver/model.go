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

package splunksearchapireceiver

// CreateJobResponse struct to represent the XML response from Splunk create job endpoint
// https://docs.splunk.com/Documentation/Splunk/9.3.1/RESTREF/RESTsearch#search.2Fjobs
type CreateJobResponse struct {
	SID string `xml:"sid"`
}

// SearchJobStatusResponse struct to represent the XML response from Splunk job status endpoint
// https://docs.splunk.com/Documentation/Splunk/9.3.1/RESTREF/RESTsearch#search.2Fjobs.2F.7Bsearch_id.7D
type SearchJobStatusResponse struct {
	Content SearchJobContent `xml:"content"`
}

// SearchJobContent struct to represent <content> elements
type SearchJobContent struct {
	Type string `xml:"type,attr"`
	Dict Dict   `xml:"dict"`
}

// Dict struct to represent <s:dict> elements
type Dict struct {
	Keys []Key `xml:"key"`
}

// Key struct to represent <s:key> elements
type Key struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
	Dict  *Dict  `xml:"dict,omitempty"`
	List  *List  `xml:"list,omitempty"`
}

// List struct to represent <s:list> elements
type List struct {
	Items []struct {
		Value string `xml:",chardata"`
	} `xml:"item"`
}

// SearchResults struct to represent the JSON response from Splunk search results endpoint
// https://docs.splunk.com/Documentation/Splunk/9.3.1/RESTREF/RESTsearch#search.2Fv2.2Fjobs.2F.7Bsearch_id.7D.2Fresults
type SearchResults struct {
	InitOffset int `json:"init_offset"`
	Results    []struct {
		Raw  string `json:"_raw"`
		Time string `json:"_time"`
	} `json:"results"`
}

// EventRecord struct stores the offset of the last event exported successfully
type EventRecord struct {
	Offset int    `json:"offset"`
	Search string `json:"search"`
}
