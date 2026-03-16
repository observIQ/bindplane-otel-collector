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

//go:build ignore

// This program generates Go types from OCSF schema.json files.
// It discovers all version directories (e.g. v1_0_0/) containing a schema.json
// and generates a schema.go in each one.
//
// Run it with: go generate ./...
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// Schema represents the top-level OCSF schema structure.
type Schema struct {
	Version   string            `json:"version"`
	Objects   map[string]Object `json:"objects"`
	Types     map[string]Type   `json:"types"`
	BaseEvent Class             `json:"base_event"`
	Classes   map[string]Class  `json:"classes"`
}

// Type represents a primitive OCSF type.
type Type struct {
	BaseType    string `json:"type"`
	Description string `json:"description"`
	Caption     string `json:"caption"`
	Regex       string `json:"regex"`
	MaxLen      *int   `json:"max_len,omitempty"`
	Range       []int  `json:"range,omitempty"`
}

// Object represents an OCSF object definition.
type Object struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Extends     string               `json:"extends"`
	Caption     string               `json:"caption"`
	Attributes  map[string]Attribute `json:"attributes"`
	Constraints Constraints          `json:"constraints"`
}

// Class represents an OCSF event class definition.
type Class struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	UID         int                  `json:"uid"`
	Extends     string               `json:"extends"`
	Category    string               `json:"category"`
	CategoryUID int                  `json:"category_uid"`
	Caption     string               `json:"caption"`
	Attributes  map[string]Attribute `json:"attributes"`
	Constraints Constraints          `json:"constraints"`
}

// Constraints represents object/class-level validation constraints.
type Constraints struct {
	AtLeastOne []string `json:"at_least_one"`
	JustOne    []string `json:"just_one"`
}

// Attribute represents a single attribute in an object or class.
type Attribute struct {
	Type        string          `json:"type"`
	Description string          `json:"description"`
	IsArray     bool            `json:"is_array"`
	Requirement string          `json:"requirement"`
	Caption     string          `json:"caption"`
	ObjectType  string          `json:"object_type"`
	Enum        map[string]Enum `json:"enum"`
	Profile     string          `json:"profile"`
}

// Enum represents an enum value.
type Enum struct {
	Description string `json:"description"`
	Caption     string `json:"caption"`
}

func main() {
	schemaEndpoints := map[string]string{
		"v1_0_0": "https://schema.ocsf.io/1.0.0/export/schema",
		"v1_1_0": "https://schema.ocsf.io/1.1.0/export/schema",
		"v1_2_0": "https://schema.ocsf.io/1.2.0/export/schema",
		"v1_3_0": "https://schema.ocsf.io/1.3.0/export/schema",
	}

	for dir, endpoint := range schemaEndpoints {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("creating directory %s: %v", dir, err)
		}

		schemaPath := filepath.Join(dir, "ocsf_schema.json")
		fmt.Printf("Downloading %s -> %s\n", endpoint, schemaPath)

		if err := downloadFile(endpoint, schemaPath); err != nil {
			log.Fatalf("downloading schema for %s: %v", dir, err)
		}

		pkgName := strings.ReplaceAll(dir, "_", "")
		fmt.Printf("Processing %s (package %s)...\n", schemaPath, pkgName)

		if err := generateForVersion(schemaPath, dir, pkgName, endpoint); err != nil {
			log.Fatalf("generating for %s: %v", schemaPath, err)
		}
	}
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url) // nolint:gosec
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	return os.WriteFile(dest, data, 0600)
}

// edgeKey uniquely identifies a field reference: the object/class that owns
// the field and the field name itself.
type edgeKey struct {
	owner string
	field string
}

func generateForVersion(schemaPath, dir, pkgName, schemaURL string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("reading schema: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parsing schema: %w", err)
	}

	// Detect which specific edges (owner, field) create cycles.
	brokenEdges := detectCyclicEdges(schema.Objects)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by ocsf-parquet-gen from %s. DO NOT EDIT.\n\n", schemaURL)
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)

	// Object type alias for circular reference fallback.
	buf.WriteString("type Object = string\n\n")

	// Generate object structs. Skip the generic "object" type since it's
	// covered by the Object type alias above.
	for _, name := range sortedKeys(schema.Objects) {
		if name == "object" {
			continue
		}
		obj := schema.Objects[name]
		attrs := filterOutProfileAttrs(obj.Attributes)
		writeParquetStruct(&buf, name, attrs, schema.Types, brokenEdges)
	}

	// Generate BaseEvent struct from the base_event definition.
	baseAttrs := filterOutProfileAttrs(schema.BaseEvent.Attributes)
	writeParquetStruct(&buf, "base_event", baseAttrs, schema.Types, brokenEdges)

	// Generate class structs with BaseEvent embedding.
	// Skip base_event since it's already generated above.
	classNames := sortedKeys(schema.Classes)
	for _, name := range classNames {
		if name == "base_event" {
			continue
		}
		cls := schema.Classes[name]
		attrs := filterOutProfileAttrs(cls.Attributes)
		writeClassStruct(&buf, name, attrs, baseAttrs, schema.Types, brokenEdges)
	}

	// Generate ClassSchemaMap.
	writeClassSchemaMap(&buf, schema.Classes, classNames)

	outPath := filepath.Join(dir, "parquet_schema.go")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		_ = os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("formatting generated code: %w\nUnformatted output written to %s", err, outPath)
	}

	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("Generated %s (%d bytes)\n", outPath, len(formatted))
	return nil
}

// detectCyclicEdges finds ALL object_t edges that participate in cycles.
// For each edge A→B, it checks whether B can reach A through the object
// reference graph. If so, the edge is part of a cycle and is replaced with
// the Object type alias. This matches AWS Security Lake's Glue table behavior.
func detectCyclicEdges(objects map[string]Object) map[edgeKey]bool {
	broken := map[edgeKey]bool{}
	for _, name := range sortedKeys(objects) {
		obj := objects[name]
		for _, fieldName := range sortedKeys(obj.Attributes) {
			attr := obj.Attributes[fieldName]
			if attr.Type != "object_t" || attr.ObjectType == "" {
				continue
			}
			target := attr.ObjectType
			if canReach(target, name, objects) {
				broken[edgeKey{owner: name, field: fieldName}] = true
			}
		}
	}
	return broken
}

// canReach returns true if 'from' can reach 'to' via object_t references.
func canReach(from, to string, objects map[string]Object) bool {
	visited := map[string]bool{}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			return true
		}
		if visited[cur] {
			continue
		}
		visited[cur] = true
		obj, ok := objects[cur]
		if !ok {
			continue
		}
		for _, attr := range obj.Attributes {
			if attr.Type == "object_t" && attr.ObjectType != "" && !visited[attr.ObjectType] {
				queue = append(queue, attr.ObjectType)
			}
		}
	}
	return false
}

// resolveGoType maps an OCSF type name to a Go type string for parquet struct fields.
func resolveGoType(typeName string, schemaTypes map[string]Type) string {
	switch typeName {
	case "integer_t":
		return "int32"
	case "long_t":
		return "int64"
	case "float_t":
		return "float64"
	case "boolean_t":
		return "bool"
	case "timestamp_t":
		return "int64"
	default:
		if t, ok := schemaTypes[typeName]; ok && t.BaseType != "" {
			return resolveGoType(t.BaseType, schemaTypes)
		}
		return "string"
	}
}

// writeParquetStruct generates a Go struct with parquet tags for an object or base_event.
func writeParquetStruct(buf *bytes.Buffer, name string, attrs map[string]Attribute, schemaTypes map[string]Type, brokenEdges map[edgeKey]bool) {
	goName := toGoName(name)
	fmt.Fprintf(buf, "type %s struct {\n", goName)
	writeStructFields(buf, name, attrs, schemaTypes, brokenEdges)
	buf.WriteString("}\n\n")
}

// writeClassStruct generates a Go struct for a class that embeds BaseEvent
// and only adds class-specific fields (those not in base_event).
func writeClassStruct(buf *bytes.Buffer, name string, attrs map[string]Attribute, baseAttrs map[string]Attribute, schemaTypes map[string]Type, brokenEdges map[edgeKey]bool) {
	goName := toGoName(name)
	fmt.Fprintf(buf, "type %s struct {\n", goName)
	fmt.Fprintf(buf, "\tBaseEvent\n")

	// Only emit fields that are NOT in the base event.
	classOnly := make(map[string]Attribute, len(attrs))
	for fieldName, attr := range attrs {
		if _, inBase := baseAttrs[fieldName]; !inBase {
			classOnly[fieldName] = attr
		}
	}

	writeStructFields(buf, name, classOnly, schemaTypes, brokenEdges)
	buf.WriteString("}\n\n")
}

// writeStructFields writes the field lines of a struct.
func writeStructFields(buf *bytes.Buffer, ownerName string, attrs map[string]Attribute, schemaTypes map[string]Type, brokenEdges map[edgeKey]bool) {
	for _, fieldName := range sortedKeys(attrs) {
		attr := attrs[fieldName]
		if attr.Profile != "" {
			continue
		}

		fieldGoName := toGoName(fieldName)
		goType := resolveFieldType(ownerName, fieldName, attr, schemaTypes, brokenEdges)

		if attr.IsArray {
			tag := fmt.Sprintf("`parquet:\"%s,optional,list\"`", fieldName)
			fmt.Fprintf(buf, "\t%s []*%s %s\n", fieldGoName, goType, tag)
		} else {
			tag := fmt.Sprintf("`parquet:\"%s,optional\"`", fieldName)
			fmt.Fprintf(buf, "\t%s *%s %s\n", fieldGoName, goType, tag)
		}
	}
}

// resolveFieldType returns the Go type name for a field, using Object for cyclic edges.
func resolveFieldType(ownerName, fieldName string, attr Attribute, schemaTypes map[string]Type, brokenEdges map[edgeKey]bool) string {
	if attr.Type == "object_t" && attr.ObjectType != "" {
		if brokenEdges[edgeKey{owner: ownerName, field: fieldName}] {
			return "Object"
		}
		return toGoName(attr.ObjectType)
	}
	if attr.Type == "json_t" {
		return "string"
	}
	return resolveGoType(attr.Type, schemaTypes)
}

// writeClassSchemaMap generates the ClassSchemaMap variable.
func writeClassSchemaMap(buf *bytes.Buffer, classes map[string]Class, classNames []string) {
	buf.WriteString("// ClassSchemaMap maps class UIDs to zero-value struct pointers for schema extraction.\n")
	buf.WriteString("var ClassSchemaMap = map[int]any{\n")
	for _, name := range classNames {
		cls := classes[name]
		goName := toGoName(name)
		fmt.Fprintf(buf, "\t%d: (*%s)(nil),\n", cls.UID, goName)
	}
	buf.WriteString("}\n")
}

// toGoName converts a snake_case OCSF name to a PascalCase Go name.
func toGoName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")

	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		upper := strings.ToUpper(part)
		if isAcronym(upper) {
			result.WriteString(upper)
		} else {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}
	return result.String()
}

func isAcronym(s string) bool {
	acronyms := map[string]bool{
		"ID": true, "UID": true, "IP": true, "URL": true,
		"HTTP": true, "DNS": true, "TCP": true, "UDP": true,
		"TLS": true, "SSL": true, "SSH": true, "API": true,
		"CVE": true, "CVSS": true, "OS": true, "CPU": true,
		"IO": true, "RDP": true, "LDAP": true, "VPN": true,
		"MAC": true, "MFA": true,
	}
	return acronyms[s]
}

// filterOutProfileAttrs returns a copy of attrs with profile-sourced attributes removed.
func filterOutProfileAttrs(attrs map[string]Attribute) map[string]Attribute {
	filtered := make(map[string]Attribute, len(attrs))
	for name, attr := range attrs {
		if attr.Profile == "" {
			filtered[name] = attr
		}
	}
	return filtered
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
