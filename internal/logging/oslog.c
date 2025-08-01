#include <os/log.h>
#include <stdlib.h>

static os_log_t logger = NULL;

void logMessage(const char* message) {
	if (logger == NULL) {
		logger = os_log_create("com.observiq.bindplane-otel-collector", "logging");
	}
	os_log(logger, "%{public}s", message);
}