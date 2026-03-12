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
	"slices"
	"sort"
	"strconv"
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

type typeConstraint struct {
	Regex  string
	MaxLen int // 0 means no limit
	Range  []int
}

// regexOverrides provides Go-compatible (RE2) replacements for OCSF type
// regexes that use PCRE-only features. Keyed by version dir, then type name.
var regexOverrides = map[string]map[string]string{
	"v1_0_0": {
		// v1.0.0 ip_t uses PCRE atomic groups; use the v1.1.0 RE2-compatible pattern instead.
		"ip_t": `((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]).){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))`,
		// v1.0.0 datetime_t regex has URL-encoded %3A instead of ':'; use the v1.1.0 pattern.
		"datetime_t": `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(Z|[\+-]\d{2}:\d{2})?$`,
	},
}

func main() {
	schemaEndpoints := map[string]string{
		"v1_0_0": "https://schema.ocsf.io/1.0.0/export/schema",
		"v1_1_0": "https://schema.ocsf.io/1.1.0/export/schema",
		"v1_2_0": "https://schema.ocsf.io/1.2.0/export/schema",
		"v1_3_0": "https://schema.ocsf.io/1.3.0/export/schema",
		"v1_4_0": "https://schema.ocsf.io/1.4.0/export/schema",
		"v1_5_0": "https://schema.ocsf.io/1.5.0/export/schema",
		"v1_6_0": "https://schema.ocsf.io/1.6.0/export/schema",
		"v1_7_0": "https://schema.ocsf.io/1.7.0/export/schema",
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

func generateForVersion(schemaPath, dir, pkgName, schemaUrl string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("reading schema: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return fmt.Errorf("parsing schema: %w", err)
	}

	var buf bytes.Buffer
	// Generate package header
	fmt.Fprintf(&buf, "// Code generated by gen.go from %s. DO NOT EDIT.\n\n", schemaUrl)
	fmt.Fprintf(&buf, "package %s\n\n", pkgName)

	typeConstraints := map[string]typeConstraint{}
	for _, t := range sortedKeys(schema.Types) {
		tc := typeConstraint{}
		if overrides, ok := regexOverrides[dir]; ok {
			if pattern, ok := overrides[t]; ok {
				tc.Regex = pattern
			}
		}
		if tc.Regex == "" && schema.Types[t].Regex != "" {
			tc.Regex = schema.Types[t].Regex
		}
		if schema.Types[t].MaxLen != nil {
			tc.MaxLen = *schema.Types[t].MaxLen
		}
		if len(schema.Types[t].Range) == 2 {
			tc.Range = schema.Types[t].Range
		}
		if tc.Regex != "" || tc.MaxLen > 0 || len(tc.Range) == 2 {
			typeConstraints[t] = tc
		}
	}

	writePackages(&buf)
	writeRegexVars(&buf, typeConstraints)
	writeHelpers(&buf)

	for _, name := range sortedKeys(schema.Objects) {
		obj := schema.Objects[name]
		generateType(&buf, name, obj.Attributes, obj.Constraints, typeConstraints)
	}

	classNames := sortedKeys(schema.Classes)
	for _, name := range classNames {
		cls := schema.Classes[name]
		generateType(&buf, name, cls.Attributes, cls.Constraints, typeConstraints)
	}

	// Write one validate function per unique profile
	writeProfileValidateFunctions(&buf, schema, typeConstraints)

	// Generate class UID constants
	buf.WriteString("// Class UIDs\n")
	buf.WriteString("const (\n")
	for _, name := range classNames {
		cls := schema.Classes[name]
		fmt.Fprintf(&buf, "ClassUID%s = %d\n", toGoName(name), cls.UID)
	}
	buf.WriteString(")\n\n")

	// Generate ValidateClass function
	writeValidateClass(&buf, schema, classNames)

	// Generate ValidateProfile function
	writeValidateProfile(&buf, schema, classNames)

	// Generate field coverage validation (config-time required field checks)
	writeFieldCoverageValidation(&buf, schema, classNames)

	// Generate Schema struct that implements the OCSFSchema interface
	writeSchemaStruct(&buf)

	outPath := filepath.Join(dir, "schema.go")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted so we can debug
		_ = os.WriteFile(outPath, buf.Bytes(), 0644)
		return fmt.Errorf("formatting generated code: %w\nUnformatted output written to %s", err, outPath)
	}

	if err := os.WriteFile(outPath, formatted, 0644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("Generated %s (%d bytes)\n", outPath, len(formatted))
	return nil
}

func generateType(buf *bytes.Buffer, name string, attrs map[string]Attribute, constraints Constraints, typeConstraints map[string]typeConstraint) {
	baseAttrs := filterOutProfileAttrs(attrs)
	writeValidation(buf, toGoName(name), baseAttrs, constraints, typeConstraints)
}

// writeProfileValidateFunctions generates one validate function per unique profile
// (e.g. validateProfileCloud, validateProfileDatetime). Profile attributes are the
// same across all classes, so we only need one function per profile.
func writeProfileValidateFunctions(buf *bytes.Buffer, schema Schema, typeConstraints map[string]typeConstraint) {
	// Collect profile attrs from across all classes — each profile defines the
	// same set of attributes regardless of which class it appears on.
	profileAttrs := map[string]map[string]Attribute{}
	for _, cls := range schema.Classes {
		for attrName, attr := range cls.Attributes {
			if attr.Profile == "" {
				continue
			}
			if profileAttrs[attr.Profile] == nil {
				profileAttrs[attr.Profile] = map[string]Attribute{}
			}
			profileAttrs[attr.Profile][attrName] = attr
		}
	}

	for _, profileName := range sortedKeys(profileAttrs) {
		funcName := fmt.Sprintf("Profile%s", toGoName(profileName))
		writeValidation(buf, funcName, profileAttrs[profileName], Constraints{}, typeConstraints)
	}
}

func writePackages(buf *bytes.Buffer) {
	stdPackages := []string{"errors", "fmt", "regexp", "strings"}

	buf.WriteString("import (\n")
	for _, pkg := range stdPackages {
		fmt.Fprintf(buf, "%q\n", pkg)
	}
	buf.WriteString(")\n\n")
}

// writeRegexVars writes precompiled regexp variables for each OCSF type that has a regex pattern.
func writeRegexVars(buf *bytes.Buffer, typeConstraints map[string]typeConstraint) {
	var hasAny bool
	for _, tc := range typeConstraints {
		if tc.Regex != "" {
			hasAny = true
			break
		}
	}
	if !hasAny {
		return
	}
	buf.WriteString("// Precompiled regex patterns for OCSF types.\n")
	buf.WriteString("var (\n")
	for _, typeName := range sortedKeys(typeConstraints) {
		tc := typeConstraints[typeName]
		if tc.Regex == "" {
			continue
		}
		varName := "regex" + toGoName(typeName)
		fmt.Fprintf(buf, "%s = regexp.MustCompile(%q)\n", varName, tc.Regex)
	}
	buf.WriteString(")\n\n")
}

// writeHelpers generates helper functions used by the validation code.
func writeHelpers(buf *bytes.Buffer) {
	buf.WriteString(`// toInt64 converts a numeric value to int64 for validation.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

`)
}

// writeValidation generates a standalone validate function that operates on map[string]any.
// Every type gets a validate function so parents can always recurse into children.
func writeValidation(buf *bytes.Buffer, goName string, attrs map[string]Attribute, constraints Constraints, typeConstraints map[string]typeConstraint) {
	fmt.Fprintf(buf, "// validate%s checks required fields, constraints, and enum values.\n", goName)
	fmt.Fprintf(buf, "func validate%s(data map[string]any) error {\n", goName)
	buf.WriteString("var errs []error\n")

	// Required field checks
	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if attr.Requirement != "required" {
			continue
		}
		fmt.Fprintf(buf, "if _, ok := data[%q]; !ok {\n", name)
		fmt.Fprintf(buf, "errs = append(errs, errors.New(%q))\n", name+" is required")
		buf.WriteString("}\n")
	}

	// at_least_one constraint
	if len(constraints.AtLeastOne) > 0 {
		fields := filterFieldsInAttrs(constraints.AtLeastOne, attrs)
		if len(fields) > 0 {
			buf.WriteString("{\n")
			conditions := make([]string, 0, len(fields))
			for i, f := range fields {
				varName := fmt.Sprintf("ok%d", i)
				fmt.Fprintf(buf, "_, %s := data[%q]\n", varName, f)
				conditions = append(conditions, "!"+varName)
			}
			fmt.Fprintf(buf, "if %s {\n", strings.Join(conditions, " && "))
			fmt.Fprintf(buf, "errs = append(errs, errors.New(\"at least one of [%s] must be set\"))\n", strings.Join(fields, ", "))
			buf.WriteString("}\n}\n")
		}
	}

	// just_one constraint
	if len(constraints.JustOne) > 0 {
		fields := filterFieldsInAttrs(constraints.JustOne, attrs)
		if len(fields) > 0 {
			buf.WriteString("{\ncount := 0\n")
			for _, f := range fields {
				fmt.Fprintf(buf, "if _, ok := data[%q]; ok { count++ }\n", f)
			}
			buf.WriteString("if count != 1 {\n")
			fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"exactly one of [%s] must be set, got %%d\", count))\n", strings.Join(fields, ", "))
			buf.WriteString("}\n}\n")
		}
	}

	// Enum validation (skip array fields — enum constraints don't apply per-element)
	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if len(attr.Enum) == 0 || attr.IsArray {
			continue
		}
		writeEnumValidation(buf, name, attr)
	}

	// Type-level validation: range, max length, and regex
	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if attr.IsArray {
			continue
		}
		tc, ok := typeConstraints[attr.Type]
		if !ok {
			continue
		}
		if tc.Range != nil {
			writeRangeValidation(buf, name, tc.Range)
		}
		if tc.MaxLen > 0 {
			writeMaxLenValidation(buf, name, tc.MaxLen)
		}
		if tc.Regex != "" {
			writeRegexValidation(buf, name, attr.Type)
		}
	}

	// Recursive validation of nested objects
	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if attr.Type != "object_t" || attr.ObjectType == "" {
			continue
		}
		nestedFn := "validate" + toGoName(attr.ObjectType)
		if attr.IsArray {
			fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", name)
			buf.WriteString("if arr, ok := v.([]any); ok {\n")
			buf.WriteString("for i, elem := range arr {\n")
			buf.WriteString("if m, ok := elem.(map[string]any); ok {\n")
			fmt.Fprintf(buf, "if err := %s(m); err != nil {\n", nestedFn)
			fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s[%%d]: %%w\", i, err))\n", name)
			buf.WriteString("}\n}\n}\n}\n}\n")
		} else {
			fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", name)
			buf.WriteString("if m, ok := v.(map[string]any); ok {\n")
			fmt.Fprintf(buf, "if err := %s(m); err != nil {\n", nestedFn)
			fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: %%w\", err))\n", name)
			buf.WriteString("}\n}\n}\n")
		}
	}

	buf.WriteString("return errors.Join(errs...)\n")
	buf.WriteString("}\n\n")
}

func writeRangeValidation(buf *bytes.Buffer, fieldName string, rangeVals []int) {
	fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", fieldName)
	buf.WriteString("if intVal, ok := toInt64(v); ok {\n")
	fmt.Fprintf(buf, "if intVal < %d || intVal > %d {\n", rangeVals[0], rangeVals[1])
	fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: value %%d is out of range [%d, %d]\", intVal))\n", fieldName, rangeVals[0], rangeVals[1])
	buf.WriteString("}\n}\n}\n")
}

func writeMaxLenValidation(buf *bytes.Buffer, fieldName string, maxLen int) {
	fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", fieldName)
	buf.WriteString("if strVal, ok := v.(string); ok {\n")
	fmt.Fprintf(buf, "if len(strVal) > %d {\n", maxLen)
	fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: length %%d exceeds max %d\", len(strVal)))\n", fieldName, maxLen)
	buf.WriteString("}\n}\n}\n")
}

// writeRegexValidation generates a regex check for a field using the precompiled type regex var.
func writeRegexValidation(buf *bytes.Buffer, fieldName string, typeName string) {
	varName := "regex" + toGoName(typeName)
	fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", fieldName)
	buf.WriteString("if strVal, ok := v.(string); ok {\n")
	fmt.Fprintf(buf, "if !%s.MatchString(strVal) {\n", varName)
	fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: invalid value %%q\", strVal))\n", fieldName)
	buf.WriteString("}\n}\n}\n")
}

// writeEnumValidation generates a switch-based enum check for a single field.
func writeEnumValidation(buf *bytes.Buffer, fieldName string, attr Attribute) {
	switch attr.Type {
	case "long_t", "integer_t":
		vals := parseIntEnumKeys(attr.Enum)
		if len(vals) == 0 {
			return
		}
		fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", fieldName)
		buf.WriteString("if intVal, ok := toInt64(v); ok {\n")
		buf.WriteString("switch intVal {\n")
		fmt.Fprintf(buf, "case %s:\n", joinInts(vals))
		buf.WriteString("default:\n")
		fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: invalid value %%d\", intVal))\n", fieldName)
		buf.WriteString("}\n}\n}\n")
	default:
		// String-like types with enums
		if !strings.HasSuffix(attr.Type, "_t") {
			return
		}
		keys := sortedKeys(attr.Enum)
		if len(keys) == 0 {
			return
		}
		quoted := make([]string, 0, len(keys))
		for _, k := range keys {
			quoted = append(quoted, fmt.Sprintf("%q", k))
		}
		fmt.Fprintf(buf, "if v, ok := data[%q]; ok {\n", fieldName)
		buf.WriteString("if strVal, ok := v.(string); ok {\n")
		buf.WriteString("switch strVal {\n")
		fmt.Fprintf(buf, "case %s:\n", strings.Join(quoted, ", "))
		buf.WriteString("default:\n")
		fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: invalid value %%q\", strVal))\n", fieldName)
		buf.WriteString("}\n}\n}\n")
	}
}

// writeValidateClass generates a ValidateClass function that validates data
// directly as map[string]any against the appropriate class, including profile validation.
func writeValidateClass(buf *bytes.Buffer, schema Schema, classNames []string) {
	buf.WriteString("// ValidateClass validates data against the OCSF event class identified by classUID.\n")
	buf.WriteString("// If profiles are provided, profile-specific validation is also applied.\n")
	buf.WriteString("func ValidateClass(classUID int, profiles []string, data any) error {\n")
	buf.WriteString("m, ok := data.(map[string]any)\n")
	buf.WriteString("if !ok {\n")
	buf.WriteString("return fmt.Errorf(\"expected map[string]any, got %T\", data)\n")
	buf.WriteString("}\n")
	buf.WriteString("var baseErr error\n")
	buf.WriteString("switch classUID {\n")
	for _, name := range classNames {
		goName := toGoName(name)
		cls := schema.Classes[name]

		// Collect profiles for this class
		classProfileSet := map[string]bool{}
		for _, attr := range cls.Attributes {
			if attr.Profile != "" {
				classProfileSet[attr.Profile] = true
			}
		}
		classProfiles := sortedKeys(classProfileSet)

		fmt.Fprintf(buf, "case ClassUID%s:\n", goName)
		fmt.Fprintf(buf, "baseErr = validate%s(m)\n", goName)

		if len(classProfiles) > 0 {
			buf.WriteString("for _, p := range profiles {\n")
			buf.WriteString("switch p {\n")
			for _, prof := range classProfiles {
				goProf := toGoName(prof)
				fmt.Fprintf(buf, "case %q:\n", prof)
				fmt.Fprintf(buf, "if err := validateProfile%s(m); err != nil {\n", goProf)
				buf.WriteString("baseErr = errors.Join(baseErr, err)\n")
				buf.WriteString("}\n")
			}
			buf.WriteString("}\n") // switch
			buf.WriteString("}\n") // for
		}
	}
	buf.WriteString("default:\n")
	buf.WriteString("return fmt.Errorf(\"unknown class UID: %d\", classUID)\n")
	buf.WriteString("}\n")
	buf.WriteString("return baseErr\n")
	buf.WriteString("}\n\n")
}

func writeValidateProfile(buf *bytes.Buffer, schema Schema, classNames []string) {
	// Generate per-class validateProfileXxx functions that check if a profile name is valid
	for _, name := range classNames {
		goName := toGoName(name)
		cls := schema.Classes[name]

		classProfileSet := map[string]bool{}
		for _, attr := range cls.Attributes {
			if attr.Profile != "" {
				classProfileSet[attr.Profile] = true
			}
		}
		classProfiles := sortedKeys(classProfileSet)

		fmt.Fprintf(buf, "func validateProfile%s(profile string) error {\n", goName)
		if len(classProfiles) > 0 {
			buf.WriteString("switch profile {\n")
			fmt.Fprintf(buf, "case %s:\n", quoteAndJoin(classProfiles))
			buf.WriteString("return nil\n")
			buf.WriteString("default:\n")
			fmt.Fprintf(buf, "return fmt.Errorf(\"profile %%q is not valid for class %s\", profile)\n", name)
			buf.WriteString("}\n")
		} else {
			fmt.Fprintf(buf, "return fmt.Errorf(\"profile %%q is not valid for class %s\", profile)\n", name)
		}
		buf.WriteString("}\n\n")
	}

	// Generate the top-level ValidateProfile dispatcher
	buf.WriteString("// ValidateProfile makes sure the profile is valid for the class identified by classUID.\n")
	buf.WriteString("func ValidateProfile(classUID int, profile string) error {\n")
	buf.WriteString("switch classUID {\n")
	for _, name := range classNames {
		goName := toGoName(name)
		fmt.Fprintf(buf, "case ClassUID%s:\n", goName)
		fmt.Fprintf(buf, "return validateProfile%s(profile)\n", goName)
	}
	buf.WriteString("default:\n")
	buf.WriteString("return fmt.Errorf(\"unknown class UID: %d\", classUID)\n")
	buf.WriteString("}\n")
	buf.WriteString("}\n\n")
}

// filterFieldsInAttrs returns only the constraint field names that exist as
// direct attributes (skips dotted paths like "device.hostname").
func filterFieldsInAttrs(fields []string, attrs map[string]Attribute) []string {
	var result []string
	for _, f := range fields {
		if strings.Contains(f, ".") {
			continue
		}
		if _, ok := attrs[f]; ok {
			result = append(result, f)
		}
	}
	sort.Strings(result)
	return result
}

// parseIntEnumKeys parses the enum map keys as integers and returns them sorted.
func parseIntEnumKeys(enums map[string]Enum) []int64 {
	vals := make([]int64, 0, len(enums))
	for k := range enums {
		v, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			continue
		}
		vals = append(vals, v)
	}
	slices.Sort(vals)
	return vals
}

func joinInts(vals []int64) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(parts, ", ")
}

// toGoName converts a snake_case OCSF name to a PascalCase Go name.
func toGoName(name string) string {
	// Handle namespaced names like "win/registry_value_query"
	name = strings.ReplaceAll(name, "/", "_")

	parts := strings.Split(name, "_")
	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Handle well-known acronyms
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
// Profile attributes come from optional OCSF overlays (e.g. "cloud", "osint", "security_control")
// and should not be included in the base class/object definition.
func filterOutProfileAttrs(attrs map[string]Attribute) map[string]Attribute {
	filtered := make(map[string]Attribute, len(attrs))
	for name, attr := range attrs {
		if attr.Profile == "" {
			filtered[name] = attr
		}
	}
	return filtered
}

// filterAttrsForProfile returns only attributes belonging to the given profile.
func filterAttrsForProfile(attrs map[string]Attribute, profile string) map[string]Attribute {
	filtered := make(map[string]Attribute)
	for name, attr := range attrs {
		if attr.Profile == profile {
			filtered[name] = attr
		}
	}
	return filtered
}

// collectAllProfiles returns sorted unique profile names from all classes and objects.
func collectAllProfiles(schema Schema) []string {
	profiles := map[string]bool{}
	for _, cls := range schema.Classes {
		for _, attr := range cls.Attributes {
			if attr.Profile != "" {
				profiles[attr.Profile] = true
			}
		}
	}
	for _, obj := range schema.Objects {
		for _, attr := range obj.Attributes {
			if attr.Profile != "" {
				profiles[attr.Profile] = true
			}
		}
	}
	result := make([]string, 0, len(profiles))
	for p := range profiles {
		result = append(result, p)
	}
	sort.Strings(result)
	return result
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeSchemaStruct generates a Schema struct type with methods that delegate to the
// package-level functions, allowing each version package to satisfy the OCSFSchema interface.
func writeSchemaStruct(buf *bytes.Buffer) {
	buf.WriteString(`// Schema implements the OCSFSchema interface for this version.
type Schema struct{}

// ValidateClass validates data against the OCSF event class identified by classUID.
func (Schema) ValidateClass(classUID int, profiles []string, data any) error {
	return ValidateClass(classUID, profiles, data)
}

// ValidateProfile makes sure the profile is valid for the class identified by classUID.
func (Schema) ValidateProfile(classUID int, profile string) error {
	return ValidateProfile(classUID, profile)
}

// LookupFieldType returns the coercion type name for a field path in the given class.
func (Schema) LookupFieldType(classUID int, profiles []string, fieldPath string) string {
	return LookupFieldType(classUID, profiles, fieldPath)
}

// ValidateFieldCoverage checks that fieldPaths cover all required fields for the class identified by classUID.
func (Schema) ValidateFieldCoverage(classUID int, profiles []string, fieldPaths []string) error {
	return ValidateFieldCoverage(classUID, profiles, fieldPaths)
}
`)
}

// writeFieldCoverageValidation generates the fieldReqs type, classFieldReqs/objectFieldReqs
// maps, and the ValidateFieldCoverage/validateCoverage/splitFirst functions for config-time
// required field validation.
func writeFieldCoverageValidation(buf *bytes.Buffer, schema Schema, classNames []string) {
	// Write fieldReqs type
	buf.WriteString("// fieldReqs describes the requirement metadata for a class or object.\n")
	buf.WriteString("type fieldReqs struct {\n")
	buf.WriteString("required     []string\n")
	buf.WriteString("objectFields map[string]string\n")
	buf.WriteString("fieldTypes   map[string]string\n")
	buf.WriteString("atLeastOne   [][]string\n")
	buf.WriteString("justOne      [][]string\n")
	buf.WriteString("}\n\n")

	// Build classFieldReqs map
	buf.WriteString("var classFieldReqs = map[int]*fieldReqs{\n")
	for _, name := range classNames {
		if name == "base_event" {
			continue
		}
		cls := schema.Classes[name]
		attrs := filterOutProfileAttrs(cls.Attributes)
		writeFieldReqsMapEntry(buf, fmt.Sprintf("ClassUID%s", toGoName(name)), attrs, cls.Constraints, schema.Types)
	}
	buf.WriteString("}\n\n")

	// Build objectFieldReqs map — include all objects so that field type
	// lookups can resolve nested paths. writeFieldReqsMapEntry skips
	// entries that have no metadata at all.
	buf.WriteString("var objectFieldReqs = map[string]*fieldReqs{\n")
	objectNames := sortedKeys(schema.Objects)
	for _, name := range objectNames {
		obj := schema.Objects[name]
		attrs := filterOutProfileAttrs(obj.Attributes)
		writeFieldReqsMapEntry(buf, fmt.Sprintf("%q", name), attrs, obj.Constraints, schema.Types)
	}
	buf.WriteString("}\n\n")

	// Build profileClassFieldReqs map — keyed by profile name, then class UID
	allProfiles := collectAllProfiles(schema)

	buf.WriteString("var profileClassFieldReqs = map[string]map[int]*fieldReqs{\n")
	for _, profile := range allProfiles {
		fmt.Fprintf(buf, "%q: {\n", profile)
		for _, name := range classNames {
			if name == "base_event" {
				continue
			}
			cls := schema.Classes[name]
			profileAttrs := filterAttrsForProfile(cls.Attributes, profile)
			if len(profileAttrs) > 0 {
				writeFieldReqsMapEntry(buf, fmt.Sprintf("ClassUID%s", toGoName(name)), profileAttrs, Constraints{}, schema.Types)
			}
		}
		buf.WriteString("},\n")
	}
	buf.WriteString("}\n\n")

	// Build validProfiles set
	buf.WriteString("var validProfiles = map[string]bool{\n")
	for _, p := range allProfiles {
		fmt.Fprintf(buf, "%q: true,\n", p)
	}
	buf.WriteString("}\n\n")

	// Write static validation functions
	writeFieldCoverageFuncs(buf)
}

// resolveCoercionType resolves an OCSF type name to a coercion type by
// following the type hierarchy. For example, port_t -> integer_t -> "integer".
func resolveCoercionType(typeName string, schemaTypes map[string]Type) string {
	switch typeName {
	case "integer_t":
		return "integer"
	case "long_t":
		return "long"
	case "float_t":
		return "float"
	case "boolean_t":
		return "boolean"
	case "timestamp_t":
		return "timestamp"
	case "datetime_t":
		return "datetime"
	case "object_t", "json_t":
		return ""
	default:
		if t, ok := schemaTypes[typeName]; ok && t.BaseType != "" {
			return resolveCoercionType(t.BaseType, schemaTypes)
		}
		return "string"
	}
}

// writeFieldReqsMapEntry writes a single entry in a fieldReqs map.
func writeFieldReqsMapEntry(buf *bytes.Buffer, key string, attrs map[string]Attribute, constraints Constraints, schemaTypes map[string]Type) {
	var required []string
	objectFields := map[string]string{}
	fieldTypes := map[string]string{}

	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if attr.Requirement == "required" {
			required = append(required, name)
		}
		if attr.Type == "object_t" && attr.ObjectType != "" {
			objectFields[name] = attr.ObjectType
		}
		// Map OCSF types to coercion type names for scalar fields.
		coercion := resolveCoercionType(attr.Type, schemaTypes)
		if coercion != "" {
			fieldTypes[name] = coercion
		}
	}

	atLeastOne := filterFieldsInAttrs(constraints.AtLeastOne, attrs)
	justOne := filterFieldsInAttrs(constraints.JustOne, attrs)

	// Skip entirely empty entries
	if len(required) == 0 && len(objectFields) == 0 && len(fieldTypes) == 0 && len(atLeastOne) == 0 && len(justOne) == 0 {
		return
	}

	fmt.Fprintf(buf, "%s: {\n", key)
	if len(required) > 0 {
		fmt.Fprintf(buf, "required: []string{%s},\n", quoteAndJoin(required))
	}
	if len(objectFields) > 0 {
		buf.WriteString("objectFields: map[string]string{")
		for _, name := range sortedKeys(objectFields) {
			fmt.Fprintf(buf, "%q: %q, ", name, objectFields[name])
		}
		buf.WriteString("},\n")
	}
	if len(fieldTypes) > 0 {
		buf.WriteString("fieldTypes: map[string]string{")
		for _, name := range sortedKeys(fieldTypes) {
			fmt.Fprintf(buf, "%q: %q, ", name, fieldTypes[name])
		}
		buf.WriteString("},\n")
	}
	if len(atLeastOne) > 0 {
		fmt.Fprintf(buf, "atLeastOne: [][]string{{%s}},\n", quoteAndJoin(atLeastOne))
	}
	if len(justOne) > 0 {
		fmt.Fprintf(buf, "justOne: [][]string{{%s}},\n", quoteAndJoin(justOne))
	}
	buf.WriteString("},\n")
}

// quoteAndJoin returns a comma-separated list of quoted strings.
func quoteAndJoin(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}

// writeFieldCoverageFuncs writes the ValidateFieldCoverage, validateCoverage,
// splitFirst, and LookupFieldType functions as static code.
func writeFieldCoverageFuncs(buf *bytes.Buffer) {
	buf.WriteString(`// ValidateFieldCoverage checks that fieldPaths cover all required fields
// for the class identified by classUID, recursively validating nested objects.
// If profiles are provided, profile-specific required fields are also checked.
// fieldPaths are dot-notation paths as configured by the user (e.g., "metadata.product.name").
func ValidateFieldCoverage(classUID int, profiles []string, fieldPaths []string) error {
	reqs, ok := classFieldReqs[classUID]
	if !ok {
		return fmt.Errorf("unknown class UID: %d", classUID)
	}

	for _, p := range profiles {
		if !validProfiles[p] {
			return fmt.Errorf("unknown profile: %q", p)
		}
	}

	err := validateCoverage(reqs, fieldPaths, "")

	for _, p := range profiles {
		if profileReqs, ok := profileClassFieldReqs[p]; ok {
			if pReqs, ok := profileReqs[classUID]; ok {
				if pErr := validateCoverage(pReqs, fieldPaths, ""); pErr != nil {
					err = errors.Join(err, pErr)
				}
			}
		}
	}

	return err
}

func validateCoverage(reqs *fieldReqs, paths []string, prefix string) error {
	var errs []error

	// Group paths by top-level key
	grouped := map[string][]string{}
	covered := map[string]bool{}
	for _, p := range paths {
		top, sub := splitFirst(p)
		covered[top] = true
		if sub != "" {
			grouped[top] = append(grouped[top], sub)
		}
	}

	// Check required fields
	for _, req := range reqs.required {
		if !covered[req] {
			errs = append(errs, fmt.Errorf("missing required field %q", prefix+req))
		}
	}

	// Check at_least_one constraints
	for _, group := range reqs.atLeastOne {
		found := false
		for _, f := range group {
			if covered[f] {
				found = true
				break
			}
		}
		if !found {
			qualifiedGroup := make([]string, len(group))
			for i, f := range group {
				qualifiedGroup[i] = prefix + f
			}
			errs = append(errs, fmt.Errorf("at least one of %v must be mapped", qualifiedGroup))
		}
	}

	// Check just_one constraints
	for _, group := range reqs.justOne {
		count := 0
		for _, f := range group {
			if covered[f] {
				count++
			}
		}
		if count != 1 {
			qualifiedGroup := make([]string, len(group))
			for i, f := range group {
				qualifiedGroup[i] = prefix + f
			}
			errs = append(errs, fmt.Errorf("exactly one of %v must be mapped, got %d", qualifiedGroup, count))
		}
	}

	// Recurse into object fields that have sub-paths
	for field, subPaths := range grouped {
		objType, ok := reqs.objectFields[field]
		if !ok {
			continue
		}
		objReqs, ok := objectFieldReqs[objType]
		if !ok {
			continue
		}
		if err := validateCoverage(objReqs, subPaths, prefix+field+"."); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func splitFirst(s string) (string, string) {
	i := strings.IndexByte(s, '.')
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}

// LookupFieldType returns the coercion type name for a field path in the
// given class. It resolves dot-notation paths (e.g. "src_endpoint.ip") by
// recursing through object field definitions. Returns "" if the field or
// class is unknown. If profiles are provided, profile-specific fields are
// also searched.
func LookupFieldType(classUID int, profiles []string, fieldPath string) string {
	reqs, ok := classFieldReqs[classUID]
	if ok {
		if t := lookupFieldTypeInReqs(reqs, fieldPath); t != "" {
			return t
		}
	}
	for _, p := range profiles {
		if profileReqs, ok := profileClassFieldReqs[p]; ok {
			if pReqs, ok := profileReqs[classUID]; ok {
				if t := lookupFieldTypeInReqs(pReqs, fieldPath); t != "" {
					return t
				}
			}
		}
	}
	return ""
}

func lookupFieldTypeInReqs(reqs *fieldReqs, path string) string {
	top, sub := splitFirst(path)
	if sub == "" {
		return reqs.fieldTypes[top]
	}
	objType, ok := reqs.objectFields[top]
	if !ok {
		return ""
	}
	objReqs, ok := objectFieldReqs[objType]
	if !ok {
		return ""
	}
	return lookupFieldTypeInReqs(objReqs, sub)
}

`)
}
