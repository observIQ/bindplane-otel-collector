package chronicleexporter

import "go.opentelemetry.io/otel/attribute"

var (
	attrError = "error"

	attrErrorNone    attribute.KeyValue = attribute.String(attrError, "none")
	attrErrorUnknown attribute.KeyValue = attribute.String(attrError, "unknown")
)
