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
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/observiq/bindplane-otel-collector/receiver/pluginreceiver"
	"gopkg.in/yaml.v3"
)

// PluginDetails gives the template enough info for the doc generation
type PluginDetails struct {
	ExtraDetails ExtraDetails
	Plugin       *pluginreceiver.Plugin
	PluginDir    string
}

// ExtraDetails holds info from a custom file for a plugin for non standard information
type ExtraDetails struct {
	Top                 string `yaml:"top"`
	Bottom              string `yaml:"bottom"`
	CustomExampleParams string `yaml:"custom_example_params"`
}

// PluginDocGenerator is responsible for generating all plugin documentation
type PluginDocGenerator struct {
	pluginDir             string
	pluginExtraDetailsDir string
	pluginDocsDir         string
	pluginDocsTemplate    *template.Template
}

// newPluginDocGenerator initializes and returns a new PluginDocGenerator
func newPluginDocGenerator(pluginDir string, pluginExtraDetailsDir string, pluginDocsDir string) *PluginDocGenerator {
	// Helper functions for template
	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"IsTypeArray": func(paramType pluginreceiver.ParameterType) bool {
			if paramType == "[]string" {
				return true
			}

			return false
		},
		"IsNotNil": func(param any) bool {
			if param == nil {
				return false
			}

			return true
		},
		"IsNotWhiteSpaceString": func(param any) bool {
			strParam, ok := param.(string)
			if !ok {
				return true
			}

			if strings.TrimSpace(strParam) == "" {
				return false
			}

			return true
		},
	}

	// Template files are provided as a slice of strings
	paths := []string{"templates/readme.tmpl"}
	pluginDocsTemplate := template.Must(template.New("readme.tmpl").Funcs(funcMap).ParseFiles(paths...))

	pdg := &PluginDocGenerator{
		pluginDir:             pluginDir,
		pluginExtraDetailsDir: pluginExtraDetailsDir,
		pluginDocsDir:         pluginDocsDir,
		pluginDocsTemplate:    pluginDocsTemplate,
	}

	return pdg
}

// GeneratePluginDocs generates the readme documenation for all plugins
func (g PluginDocGenerator) GeneratePluginDocs() {
	// Get plugin files from plugin directory
	pluginFiles, err := os.ReadDir(g.pluginDir)
	if err != nil {
		log.Fatalln("Failed to read plugin directory", err)
	}

	// Loop through each plugin file
	for _, pluginFile := range pluginFiles {
		g.generatePluginDoc(pluginFile)
	}
}

// generatePluginDoc generates a single plugin's readme documentation
func (g PluginDocGenerator) generatePluginDoc(pluginFile fs.DirEntry) {
	pluginFileName := pluginFile.Name()

	// Get the plugin file path
	pluginFilePath, err := filepath.Abs(filepath.Join(g.pluginDir, pluginFileName))
	if err != nil {
		log.Fatalln("Failed to determine path of plugin file", pluginFileName, ":", err)
	}
	fmt.Printf("Creating doc for %s\n", pluginFilePath)

	// Load the plugin file so we have a plugin to work with
	plugin, err := pluginreceiver.LoadPlugin(pluginFilePath)
	if err != nil {
		log.Fatalln("Failed to load plugin", pluginFileName, ":", err)
	}

	// Get the readme file path for this plugin
	pluginReadmeName := strings.TrimSuffix(pluginFileName, filepath.Ext(pluginFileName)) + ".md"
	pluginReadmePath, err := filepath.Abs(filepath.Join(g.pluginDocsDir, pluginReadmeName))
	if err != nil {
		log.Fatalln("Failed to determine path of plugin doc file", pluginReadmeName, ":", err)
	}

	// Create readme file for this plugin
	pluginReadmeFile, err := os.Create(filepath.Clean(pluginReadmePath))
	defer func() {
		if err := pluginReadmeFile.Close(); err != nil {
			log.Fatalln("Failed to close plugin doc file", pluginReadmeName, ":", err)
		}
	}()
	if err != nil {
		log.Fatalln("Failed to create plugin doc file", pluginReadmeName, ":", err)
	}

	// Get extra details file path related to this plugin
	pluginExtraDetailsName := strings.TrimSuffix(pluginFileName, filepath.Ext(pluginFileName)) + ".yaml"
	pluginExtraDetailsPath, err := filepath.Abs(filepath.Join(g.pluginExtraDetailsDir, pluginExtraDetailsName))
	if err != nil {
		log.Fatalln("Failed to determine path of plugin extra details file", pluginExtraDetailsName, ":", err)
	}

	// If extra details file for the plugin exists, then grab that file data
	var extraDetails ExtraDetails
	if _, err := os.Stat(pluginExtraDetailsPath); errors.Is(err, os.ErrNotExist) {
		extraDetails = ExtraDetails{}
	} else {
		detailsBytes, err := ioutil.ReadFile(filepath.Clean(pluginExtraDetailsPath))
		if err != nil {
			extraDetails = ExtraDetails{}
		} else if err = yaml.Unmarshal(detailsBytes, &extraDetails); err != nil {
			log.Fatalln("Failed to unmarshal plugin extra details", pluginExtraDetailsName, ":", err)
		}
	}

	// Create struct for template execution
	pluginDetails := PluginDetails{
		Plugin:       plugin,
		ExtraDetails: extraDetails,
		PluginDir:    strings.Join([]string{"./plugins", pluginFileName}, "/"),
	}

	// Populate plugin readme doc
	err = g.pluginDocsTemplate.Execute(pluginReadmeFile, pluginDetails)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Doc created for %s\n\n", pluginFilePath)
}
