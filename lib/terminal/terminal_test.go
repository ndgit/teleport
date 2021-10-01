// Copyright 2021 Gravitational, Inc
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

package terminal_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gravitational/teleport/lib/terminal"

	"github.com/stretchr/testify/require"
)

func TestStart(t *testing.T) {
	cfg := terminal.Config{
		Addr:    fmt.Sprintf("unix://%v/terminal.sock", t.TempDir()),
		HomeDir: fmt.Sprintf("%v/", t.TempDir()),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wait := make(chan error)
	go func() {
		err := terminal.Start(ctx, cfg)
		wait <- err
	}()

	// cc, err := grpc.Dial(cfg.Addr, grpc.WithInsecure())
	// require.NoError(t, err)

	// term := v1.NewTerminalServiceClient(cc)
	// _, err = term.CreateCluster(ctx, &v1.CreateClusterRequest{Name: "platform.teleport.sh"})
	// require.NoError(t, err)

	defer func() {
		cancel() // Stop the server.
		require.NoError(t, <-wait)
	}()

}
