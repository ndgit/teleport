/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package daemon

import (
	"context"

	"github.com/gravitational/teleport/api/client/webclient"
	"github.com/gravitational/trace"
	"github.com/jonboulle/clockwork"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/client"
)

// Cluster describes user settings and access to various resources.
type Cluster struct {
	// Name is the cluster name
	Name string
	// dir is the directory where cluster certificates are stored
	dir string
	// Status is the cluster status
	status client.ProfileStatus
	// client is the cluster Teleport client
	client *client.TeleportClient
	// clock is a clock for time-related operations
	clock clockwork.Clock
}

// Connected indicates if connection to the cluster can be established
func (c *Cluster) Connected() bool {
	return c.status.Name != "" && !c.status.IsExpired(c.clock)
}

// SyncAuthPreference fetches Teleport auth preferences and stores it in the cluster profile
func (c *Cluster) SyncAuthPreference(ctx context.Context) (*webclient.AuthenticationSettings, error) {
	pingResponse, err := c.client.Ping(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	if err := c.client.SaveProfile(c.dir, false); err != nil {
		return nil, trace.Wrap(err)
	}

	return &pingResponse.Auth, nil
}

// GetRoles returns currently logged-in user roles
func (c *Cluster) GetRoles(ctx context.Context) ([]*types.Role, error) {
	proxyClient, err := c.client.ConnectToProxy(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer proxyClient.Close()

	roles := []*types.Role{}
	for _, name := range c.status.Roles {
		role, err := proxyClient.GetRole(ctx, name)
		if err != nil {
			return nil, trace.Wrap(err)
		}
		roles = append(roles, &role)
	}

	return roles, nil
}

// GetUser returns currently logged-in user
func (c *Cluster) GetUser(ctx context.Context) (types.User, error) {
	proxyClient, err := c.client.ConnectToProxy(ctx)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	defer proxyClient.Close()

	user, err := proxyClient.GetUser(ctx, c.status.Username)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return user, nil
}

// SSOLogin logs in a user to the Teleport cluster using supported SSO provider
func (c *Cluster) SSOLogin(ctx context.Context, providerType, providerName string) error {
	// ping Teleport proxy to update this cluster profile
	if _, err := c.client.Ping(ctx); err != nil {
		return trace.Wrap(err)
	}

	key, err := c.client.SSOLogin(ctx, providerType, providerName)
	if err != nil {
		return trace.Wrap(err)
	}

	if err := c.client.ActivateKey(ctx, key); err != nil {
		return trace.Wrap(err)
	}

	if err := c.client.SaveProfile(c.dir, true); err != nil {
		return trace.Wrap(err)
	}

	return nil
}
