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

package daemon

import (
	"context"
	"fmt"

	"github.com/gravitational/trace"
	"github.com/jonboulle/clockwork"

	"github.com/gravitational/teleport/api/profile"
	"github.com/gravitational/teleport/lib/client"
)

// Config is the cluster service config
type Config struct {
	// Dir is the directory to store cluster profiles
	Dir string
	// Clock is a clock for time-related operations
	Clock clockwork.Clock
}

// CheckAndSetDefaults checks the configuration for its validity and sets default values if needed
func (c *Config) CheckAndSetDefaults() error {
	if c.Dir == "" {
		return trace.BadParameter("missing working directory")
	}

	if c.Clock == nil {
		c.Clock = clockwork.NewRealClock()
	}

	return nil
}

// Service is the cluster service
type Service struct {
	Config
	clusters []*Cluster
}

// Start creates and starts a Teleport Terminal service.
func New(cfg Config) (*Service, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	return &Service{Config: cfg}, nil
}

// GetClusters returns a list of existing clusters
func (s *Service) GetClusters() []*Cluster {
	return s.clusters
}

// CreateCluster creates a new cluster
func (s *Service) CreateCluster(ctx context.Context, clusterName string) error {
	fmt.Print("MAMA", len(s.clusters))
	for _, cluster := range s.clusters {
		fmt.Print("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", s.clusters)
		if cluster.Name == clusterName {
			return trace.BadParameter("cluster %v already exists", clusterName)
		}
	}

	cluster, err := s.newCluster(s.Dir, clusterName)
	if err != nil {
		return trace.Wrap(err)
	}

	s.clusters = append(s.clusters, cluster)
	return nil
}

// GetCluster returns a cluster by its name
func (s *Service) GetCluster(name string) (*Cluster, error) {
	for _, cluster := range s.clusters {
		if cluster.Name == name {
			return cluster, nil
		}
	}

	return nil, trace.NotFound("cluster %v is not found", name)
}

// LoadClusters initializes existing clusters from their profiles
func (s *Service) LoadClusters() error {
	pfNames, err := profile.ListProfileNames(s.Dir)
	if err != nil {
		return trace.Wrap(err)
	}

	for _, name := range pfNames {
		cluster, err := s.LoadClusterFromProfile(name)
		if err != nil {
			return trace.Wrap(err)
		}

		fmt.Printf("CLUSTER: %v %v", name, cluster)

		s.clusters = append(s.clusters, cluster)
	}

	return nil
}

// newClusterFromProfile creates new cluster from its profile
func (s *Service) LoadClusterFromProfile(name string) (*Cluster, error) {
	if name == "" {
		return nil, trace.BadParameter("name is missing")
	}

	cfg := client.MakeDefaultConfig()
	if err := cfg.LoadProfile(s.Dir, name); err != nil {
		return nil, trace.Wrap(err)
	}

	cfg.KeysDir = s.Dir
	cfg.HomePath = s.Dir

	clt, err := client.NewClient(cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	status := &client.ProfileStatus{}

	// load profile status if it exists
	_, err = clt.LocalAgent().GetKey(name)
	if err == nil || cfg.Username == "" {
		status, err = client.StatusFromFile(s.Dir, name)
		if err := clt.LoadKeyForCluster(name); err != nil {
			return nil, trace.Wrap(err)
		}
	}
	if err != nil && !trace.IsNotFound(err) {
		return nil, trace.Wrap(err)
	}

	return &Cluster{
		dir:    s.Dir,
		Name:   cfg.SiteName,
		client: clt,
		clock:  s.Clock,
		status: *status,
	}, nil
}

// newCluster creates new cluster
func (s *Service) newCluster(dir, name string) (*Cluster, error) {
	if name == "" {
		return nil, trace.BadParameter("cluster name is missing")
	}

	if dir == "" {
		return nil, trace.BadParameter("cluster directory is missing")
	}

	cfg := client.MakeDefaultConfig()
	cfg.WebProxyAddr = name
	cfg.HomePath = s.Dir
	cfg.KeysDir = s.Dir

	if err := cfg.SaveProfile(s.Dir, false); err != nil {
		return nil, trace.Wrap(err)
	}

	client, err := client.NewClient(cfg)
	if err != nil {
		return nil, trace.Wrap(err)
	}

	return &Cluster{
		dir:    s.Dir,
		Name:   name,
		client: client,
		clock:  s.Clock,
	}, nil
}
