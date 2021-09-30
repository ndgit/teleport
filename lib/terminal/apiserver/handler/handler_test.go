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

package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gravitational/teleport/lib/terminal/daemon"

	"github.com/stretchr/testify/require"
)

const (
	profileDir = "/home/alexey/go/src/github.com/gravitational/_terminal"
)

func FTestStart(t *testing.T) {
	d, err := daemon.New(daemon.Config{
		Dir: profileDir,
	})
	require.NoError(t, err)

	err = d.LoadClusters()
	require.NoError(t, err)

	err = d.CreateCluster(context.TODO(), "localhost:3080")
	require.NoError(t, err)

	// err = d.CreateCluster(context.TODO(), "platform.teleport.sh")
	// require.NoError(t, err)

	//_, err = d.GetCluster("localhost:3080")
	//require.NoError(t, err)

	//cluster.SSOLogin(context.Background(), "github", "github")

	//roles, _ := cluster.GetRoles(context.TODO())
	//fmt.Print("AAAAAAAAAAAAAAAAAAAAAA:", roles)
	//require.Error(t, err)

	// 	err = cluster.SSOLogin(context.TODO(), "saml", "okta")
	// 	require.NoError(t, err)

	// 	err = cluster2.SSOLogin(context.TODO(), "saml", "okta")
	// 	require.NoError(t, err)
}

func TestS(t *testing.T) {
	d, err := daemon.New(daemon.Config{
		Dir: profileDir,
	})
	require.NoError(t, err)

	err = d.LoadClusters()
	require.NoError(t, err)
	//
	//err = d.CreateCluster(context.TODO(), "test.sh")
	//require.NoError(t, err)

	//_, err = d.LoadClusterFromProfile("teleport.teleportinfra.sh")
	cluster, err := d.LoadClusterFromProfile("localhost:3080")

	fmt.Print("AAA", cluster.Name)
	require.Error(t, err)

	//fmt.Print("AAL CLUSTERS:", cluster.Connected())
	//require.NoError(t, err)

	//_, err = cluster.GetRoles(context.TODO())
	//require.NoError(t, err)

	//err = d.CreateCluster(context.TODO(), "platform.teleport2.sh")
	//require.NoError(t, err)
	//cluster, err := d.GetCluster("platform.teleport2.sh")
	//require.NoError(t, err)
	//_, err = cluster.GetRoles(context.TODO())
	//require.NoError(t, err)
}
