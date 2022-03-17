#!/bin/sh
# Copyright  observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


CONFIG_FILE="/testdata/netflowv5.yaml"
LOG_FILE="/testdata/stanza.log"
STDOUT_FILE="/testdata/stdout.log"
OUTPUT_FILE="/testdata/out.log"

# clear the log if it exists, is is crucial that each test
# starts with empty files
> "${LOG_FILE}"
> "${STDOUT_FILE}"
> "${OUTPUT_FILE}"

chmod 0666 $LOG_FILE
chmod 0666 $STDOUT_FILE
chmod 0666 $OUTPUT_FILE

/stanza_home/stanza \
    --config "${CONFIG_FILE}" \
    --log_file "${LOG_FILE}" >"${STDOUT_FILE}" 2>&1
