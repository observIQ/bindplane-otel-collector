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

# Runs every install_unix.sh test in this directory. Exits non-zero if any fail.
dir=$(dirname "$0")
rc=0
for t in "$dir"/test_*.sh; do
  echo "== ${t##*/}"
  sh "$t" "$@" || rc=1
done
exit $rc
