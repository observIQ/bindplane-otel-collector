package internal // import "github.com/observiq/bindplane-otel-collector/extension/badgerextension/internal"

const (
	ExtensionKey = "extension"
	// Storage class attributes
	StorageTypeAttribute = "storage_type"
	StorageTypeLSM       = "lsm"
	StorageTypeValueLog  = "value_log"
	// Operation type attributes
	OperationTypeAttribute = "operation_type"
	OperationTypeGet       = "get"
	OperationTypeSet       = "set"
	OperationTypeDelete    = "delete"
)
