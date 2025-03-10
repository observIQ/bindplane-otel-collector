// Copyright Bindplane, Inc.
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

// Package main provides entry point for the plugin docu generator
package main

import (
	"github.com/spf13/pflag"
)

func main() {
	// Parse flags
	pluginDir := pflag.String("plugins", "../../plugins", "The directory containing plugins")
	pluginExtraDetailsDir := pflag.String("extra_details", "./extra_details", "The directory containing extra details for plugin documentation")
	pluginDocsDir := pflag.String("docs", "../../docs/plugins", "The directory containing plugin documentation")
	pflag.Parse()

	generator := newPluginDocGenerator(*pluginDir, *pluginExtraDetailsDir, *pluginDocsDir)

	generator.GeneratePluginDocs()
}
