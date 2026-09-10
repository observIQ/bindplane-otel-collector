#!/usr/bin/env bash
# Copyright observIQ, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# retry.sh — run a command, retrying only on transient network/registry failures.
#
# Absorbs the CI flakes tracked in PIPE-1058 (Go module-proxy blips:
# proxy.golang.org HTTP/2 stream errors, i/o timeouts) and PIPE-1092
# (Docker Hub pull timeouts to registry-1.docker.io). Bounded, so a persistent
# outage still fails the job after RETRY_MAX attempts.
#
# By default it retries ONLY when the command output matches a known transient
# signature, so wrapping a build/test/install step never masks a real compile
# or test failure. Set RETRY_ANY=1 to retry on any non-zero exit.
#
# Usage: bash .github/scripts/retry.sh <command> [args...]
# Env:   RETRY_MAX (default 3), RETRY_DELAY base backoff seconds (default 10),
#        RETRY_ANY (default 0).
set -uo pipefail

max="${RETRY_MAX:-3}"
base_delay="${RETRY_DELAY:-10}"
retry_any="${RETRY_ANY:-0}"

# Signatures of transient network/registry failures that clear on a retry.
transient_re='proxy\.golang\.org|sum\.golang\.org|registry-1\.docker\.io|registry\.docker\.io|i/o timeout|TLS handshake timeout|connection reset|connection refused|unexpected EOF|stream error|INTERNAL_ERROR|net/http: request canceled|Client\.Timeout exceeded|Service Unavailable|Bad Gateway|Gateway Time-?out|Too Many Requests|temporary failure|no such host'

attempt=1
while true; do
  log="$(mktemp)"
  "$@" 2>&1 | tee "$log"
  status="${PIPESTATUS[0]}"
  if [ "$status" -eq 0 ]; then
    rm -f "$log"
    exit 0
  fi

  if [ "$attempt" -ge "$max" ]; then
    echo "retry: '$*' failed after ${max} attempt(s) (exit ${status}); giving up." >&2
    rm -f "$log"
    exit "$status"
  fi

  if [ "$retry_any" != "1" ] && ! grep -qiE "$transient_re" "$log"; then
    echo "retry: '$*' failed (exit ${status}) with no transient-network signature; not retrying." >&2
    rm -f "$log"
    exit "$status"
  fi
  rm -f "$log"

  delay=$(( base_delay * attempt ))
  echo "retry: transient failure on attempt ${attempt}/${max} (exit ${status}); retrying '$*' in ${delay}s..." >&2
  sleep "$delay"
  attempt=$(( attempt + 1 ))
done
