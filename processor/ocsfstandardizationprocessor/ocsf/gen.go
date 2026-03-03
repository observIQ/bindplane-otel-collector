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
// An empty string means "skip regex validation for this type entirely".
var regexOverrides = map[string]map[string]string{
	"v1_0_0": {
		// v1.0.0 ip_t uses PCRE atomic groups; use the v1.1.0 RE2-compatible pattern instead.
		"ip_t": `((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]).){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))`,
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

	for _, name := range sortedKeys(schema.Objects) {
		obj := schema.Objects[name]
		generateType(&buf, name, obj.Caption, obj.Description, 0, obj.Attributes, obj.Constraints, typeConstraints)
	}

	classNames := sortedKeys(schema.Classes)
	for _, name := range classNames {
		cls := schema.Classes[name]
		generateType(&buf, name, cls.Caption, cls.Description, cls.UID, cls.Attributes, cls.Constraints, typeConstraints)
	}

	// Generate class UID constants
	buf.WriteString("// Class UIDs\n")
	buf.WriteString("const (\n")
	for _, name := range classNames {
		cls := schema.Classes[name]
		fmt.Fprintf(&buf, "ClassUID%s = %d\n", toGoName(name), cls.UID)
	}
	buf.WriteString(")\n\n")

	// Generate ValidateClass function
	writeValidateClass(&buf, classNames)

	// Generate field coverage validation (config-time required field checks)
	writeFieldCoverageValidation(&buf, schema, classNames)

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

func generateType(buf *bytes.Buffer, name, caption, description string, uid int, attrs map[string]Attribute, constraints Constraints, typeConstraints map[string]typeConstraint) {
	attrs = filterOutProfileAttrs(attrs)
	writeStruct(buf, name, caption, description, uid, attrs)
	writeValidation(buf, toGoName(name), attrs, constraints, typeConstraints)
}

func writePackages(buf *bytes.Buffer) {
	stdPackages := []string{"errors", "fmt", "regexp"}
	stdPackages = append(stdPackages, "strings")
	externPackages := []string{"github.com/go-viper/mapstructure/v2"}

	buf.WriteString("import (\n")
	for _, pkg := range stdPackages {
		fmt.Fprintf(buf, "%q\n", pkg)
	}
	fmt.Fprintf(buf, "\n")
	for _, pkg := range externPackages {
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

func writeStruct(buf *bytes.Buffer, name, caption, description string, uid int, attrs map[string]Attribute) {
	goName := toGoName(name)
	if uid > 0 {
		fmt.Fprintf(buf, "// %s represents the OCSF %s event class (UID: %d).\n", goName, caption, uid)
	} else {
		fmt.Fprintf(buf, "// %s represents the OCSF %s object.\n", goName, caption)
	}
	if description != "" {
		fmt.Fprintf(buf, "// %s\n", cleanDescription(description))
	}
	fmt.Fprintf(buf, "type %s struct {\n", goName)
	writeFields(buf, attrs)
	buf.WriteString("}\n\n")
}

func writeFields(buf *bytes.Buffer, attrs map[string]Attribute) {
	names := sortedKeys(attrs)
	for _, name := range names {
		attr := attrs[name]
		goFieldName := toGoName(name)
		goType := resolveGoType(attr)
		jsonTag := name

		omitempty := attr.Requirement != "required"
		tag := fmt.Sprintf("`mapstructure:\"%s", jsonTag)
		if omitempty {
			tag += ",omitempty"
		}
		tag += "\"`"

		fmt.Fprintf(buf, "%s %s %s\n", goFieldName, goType, tag)
	}
}

// writeValidation generates a Validate() error method for a struct.
// Every type gets a Validate() method so parents can always recurse into children.
func writeValidation(buf *bytes.Buffer, goName string, attrs map[string]Attribute, constraints Constraints, typeConstraints map[string]typeConstraint) {
	fmt.Fprintf(buf, "// Validate checks required fields, constraints, and enum values for %s.\n", goName)
	fmt.Fprintf(buf, "func (o *%s) Validate() error {\n", goName)
	buf.WriteString("var errs []error\n")

	// Required field checks
	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if attr.Requirement != "required" {
			continue
		}
		goField := toGoName(name)
		if attr.IsArray {
			fmt.Fprintf(buf, "if len(o.%s) == 0 {\n", goField)
		} else if attr.Type == "json_t" {
			fmt.Fprintf(buf, "if o.%s == nil {\n", goField)
		} else {
			fmt.Fprintf(buf, "if o.%s == nil {\n", goField)
		}
		fmt.Fprintf(buf, "errs = append(errs, errors.New(\"%s is required\"))\n", name)
		buf.WriteString("}\n")
	}

	// at_least_one constraint
	if len(constraints.AtLeastOne) > 0 {
		fields := filterFieldsInAttrs(constraints.AtLeastOne, attrs)
		if len(fields) > 0 {
			conditions := make([]string, 0, len(fields))
			for _, f := range fields {
				attr := attrs[f]
				if attr.IsArray {
					conditions = append(conditions, fmt.Sprintf("len(o.%s) == 0", toGoName(f)))
				} else {
					conditions = append(conditions, fmt.Sprintf("o.%s == nil", toGoName(f)))
				}
			}
			fmt.Fprintf(buf, "if %s {\n", strings.Join(conditions, " && "))
			fmt.Fprintf(buf, "errs = append(errs, errors.New(\"at least one of [%s] must be set\"))\n", strings.Join(fields, ", "))
			buf.WriteString("}\n")
		}
	}

	// just_one constraint
	if len(constraints.JustOne) > 0 {
		fields := filterFieldsInAttrs(constraints.JustOne, attrs)
		if len(fields) > 0 {
			buf.WriteString("{\ncount := 0\n")
			for _, f := range fields {
				attr := attrs[f]
				if attr.IsArray {
					fmt.Fprintf(buf, "if len(o.%s) > 0 { count++ }\n", toGoName(f))
				} else {
					fmt.Fprintf(buf, "if o.%s != nil { count++ }\n", toGoName(f))
				}
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
		writeEnumValidation(buf, name, toGoName(name), attr)
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
			writeRangeValidation(buf, name, toGoName(name), tc.Range)
		}
		if tc.MaxLen > 0 {
			writeMaxLenValidation(buf, name, toGoName(name), tc.MaxLen)
		}
		if tc.Regex != "" {
			writeRegexValidation(buf, name, toGoName(name), attr.Type)
		}
	}

	// Recursive validation of nested objects
	for _, name := range sortedKeys(attrs) {
		attr := attrs[name]
		if attr.Type != "object_t" || attr.ObjectType == "" {
			continue
		}
		goField := toGoName(name)
		if attr.IsArray {
			fmt.Fprintf(buf, "for i := range o.%s {\n", goField)
			fmt.Fprintf(buf, "if err := o.%s[i].Validate(); err != nil {\n", goField)
			fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s[%%d]: %%w\", i, err))\n", name)
			buf.WriteString("}\n}\n")
		} else {
			fmt.Fprintf(buf, "if o.%s != nil {\n", goField)
			fmt.Fprintf(buf, "if err := o.%s.Validate(); err != nil {\n", goField)
			fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: %%w\", err))\n", name)
			buf.WriteString("}\n}\n")
		}
	}

	buf.WriteString("return errors.Join(errs...)\n")
	buf.WriteString("}\n\n")
}

func writeRangeValidation(buf *bytes.Buffer, fieldName, goField string, rangeVals []int) {
	fmt.Fprintf(buf, "if o.%s != nil && (*o.%s < %d || *o.%s > %d) {\n", goField, goField, rangeVals[0], goField, rangeVals[1])
	fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: value %%d is out of range [%d, %d]\", *o.%s))\n", fieldName, rangeVals[0], rangeVals[1], goField)
	buf.WriteString("}\n")
}

func writeMaxLenValidation(buf *bytes.Buffer, fieldName, goField string, maxLen int) {
	fmt.Fprintf(buf, "if o.%s != nil && len(*o.%s) > %d {\n", goField, goField, maxLen)
	fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: length %%d exceeds max %d\", len(*o.%s)))\n", fieldName, maxLen, goField)
	buf.WriteString("}\n")
}

// writeRegexValidation generates a regex check for a field using the precompiled type regex var.
func writeRegexValidation(buf *bytes.Buffer, fieldName, goField string, typeName string) {
	varName := "regex" + toGoName(typeName)
	fmt.Fprintf(buf, "if o.%s != nil && !%s.MatchString(*o.%s) {\n", goField, varName, goField)
	fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: invalid value %%q\", *o.%s))\n", fieldName, goField)
	buf.WriteString("}\n")
}

// writeEnumValidation generates a switch-based enum check for a single field.
func writeEnumValidation(buf *bytes.Buffer, fieldName, goField string, attr Attribute) {
	switch attr.Type {
	case "long_t", "integer_t":
		vals := parseIntEnumKeys(attr.Enum)
		if len(vals) == 0 {
			return
		}
		fmt.Fprintf(buf, "if o.%s != nil {\n", goField)
		fmt.Fprintf(buf, "switch *o.%s {\n", goField)
		fmt.Fprintf(buf, "case %s:\n", joinInts(vals))
		buf.WriteString("default:\n")
		fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: invalid value %%d\", *o.%s))\n", fieldName, goField)
		buf.WriteString("}\n}\n")
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
		fmt.Fprintf(buf, "if o.%s != nil {\n", goField)
		fmt.Fprintf(buf, "switch *o.%s {\n", goField)
		fmt.Fprintf(buf, "case %s:\n", strings.Join(quoted, ", "))
		buf.WriteString("default:\n")
		fmt.Fprintf(buf, "errs = append(errs, fmt.Errorf(\"%s: invalid value %%q\", *o.%s))\n", fieldName, goField)
		buf.WriteString("}\n}\n")
	}
}

// writeValidateClass generates a ValidateClass function that decodes data into
// the appropriate class struct based on classUID and validates it.
func writeValidateClass(buf *bytes.Buffer, classNames []string) {
	buf.WriteString("// ValidateClass decodes data into the OCSF event class identified by classUID\n")
	buf.WriteString("// and runs validation on the resulting struct.\n")
	buf.WriteString("func ValidateClass(classUID int, data any) error {\n")
	buf.WriteString("switch classUID {\n")
	for _, name := range classNames {
		goName := toGoName(name)
		fmt.Fprintf(buf, "case ClassUID%s:\n", goName)
		fmt.Fprintf(buf, "var obj %s\n", goName)
		buf.WriteString("if err := mapstructure.Decode(data, &obj); err != nil {\n")
		buf.WriteString("return fmt.Errorf(\"decoding class: %w\", err)\n")
		buf.WriteString("}\n")
		buf.WriteString("return obj.Validate()\n")
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

func resolveGoType(attr Attribute) string {
	var baseType string

	switch attr.Type {
	case "object_t":
		if attr.ObjectType != "" {
			baseType = "*" + toGoName(attr.ObjectType)
		} else {
			baseType = "map[string]any"
		}
	case "integer_t":
		baseType = "*int"
	case "long_t":
		baseType = "*int64"
	case "float_t":
		baseType = "*float64"
	case "boolean_t":
		baseType = "*bool"
	case "json_t":
		baseType = "any"
	case "timestamp_t":
		baseType = "*int64"
	case "port_t":
		baseType = "*int"
	default:
		// All string-like types: string_t, email_t, hostname_t, ip_t,
		// mac_t, url_t,datetime_t, file_hash_t,
		// file_name_t, file_path_t, process_name_t, resource_uid_t,
		// subnet_t username_t, uuid_t, bytestring_t, reg_key_path_t
		baseType = "*string"
	}

	if attr.IsArray {
		// Arrays use slice types; no pointer needed for the element
		baseType = strings.TrimPrefix(baseType, "*")
		return "[]" + baseType
	}

	return baseType
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

// cleanDescription strips HTML tags and normalizes whitespace for Go comments.
func cleanDescription(s string) string {
	// Strip HTML tags
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// Collapse whitespace and trim
	cleaned := strings.Join(strings.Fields(result.String()), " ")
	return cleaned
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

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
		writeFieldReqsMapEntry(buf, fmt.Sprintf("ClassUID%s", toGoName(name)), attrs, cls.Constraints)
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
		writeFieldReqsMapEntry(buf, fmt.Sprintf("%q", name), attrs, obj.Constraints)
	}
	buf.WriteString("}\n\n")

	// Write static validation functions
	writeFieldCoverageFuncs(buf)
}

// writeFieldReqsMapEntry writes a single entry in a fieldReqs map.
func writeFieldReqsMapEntry(buf *bytes.Buffer, key string, attrs map[string]Attribute, constraints Constraints) {
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
		switch attr.Type {
		case "integer_t":
			fieldTypes[name] = "integer"
		case "long_t":
			fieldTypes[name] = "long"
		case "float_t":
			fieldTypes[name] = "float"
		case "boolean_t":
			fieldTypes[name] = "boolean"
		case "object_t", "json_t":
			// Non-scalar — skip
		default:
			fieldTypes[name] = "string"
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
// and splitFirst functions as static code.
func writeFieldCoverageFuncs(buf *bytes.Buffer) {
	buf.WriteString(`// ValidateFieldCoverage checks that fieldPaths cover all required fields
// for the class identified by classUID, recursively validating nested objects.
// fieldPaths are dot-notation paths as configured by the user (e.g., "metadata.product.name").
func ValidateFieldCoverage(classUID int, fieldPaths []string) error {
	reqs, ok := classFieldReqs[classUID]
	if !ok {
		return fmt.Errorf("unknown class UID: %d", classUID)
	}
	return validateCoverage(reqs, fieldPaths, "")
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
// class is unknown.
func LookupFieldType(classUID int, fieldPath string) string {
	reqs, ok := classFieldReqs[classUID]
	if !ok {
		return ""
	}
	return lookupFieldTypeInReqs(reqs, fieldPath)
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
