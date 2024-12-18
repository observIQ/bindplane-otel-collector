// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chronicleexporter

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Override for testing
var tokenSource = func(ctx context.Context, cfg *Config) (oauth2.TokenSource, error) {
	creds, err := googleCredentials(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return creds.TokenSource, nil
}

func googleCredentials(ctx context.Context, cfg *Config) (*google.Credentials, error) {
	scope := grpcScope
	if cfg.Protocol == protocolHTTPS {
		scope = httpScope
	}
	switch {
	case cfg.Creds != "":
		return google.CredentialsFromJSON(ctx, []byte(cfg.Creds), scope)
	case cfg.CredsFilePath != "":
		credsData, err := os.ReadFile(cfg.CredsFilePath)
		if err != nil {
			return nil, fmt.Errorf("read credentials file: %w", err)
		}

		if len(credsData) == 0 {
			return nil, errors.New("credentials file is empty")
		}

		return google.CredentialsFromJSON(ctx, credsData, scope)
	default:
		return google.FindDefaultCredentials(ctx, scope)
	}
}
