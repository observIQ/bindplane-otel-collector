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

//go:build aix

package collector

// skippedFeatureGates are gates registered by components the AIX build leaves out.
// A listed gate is skipped only while the registry doesn't know it, so adding the
// component back to the AIX build puts the gate back in effect.
var skippedFeatureGates = []string{
	"exporter.prometheusremotewritexporter.enableSendingRW2", // prometheusremotewriteexporter
	"exporter.prometheusexporter.DisableAddMetricSuffixes",   // prometheusexporter
}
