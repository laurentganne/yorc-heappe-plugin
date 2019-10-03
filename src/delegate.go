// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package main

import (
	"context"
	"strings"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/tasks"
	"github.com/ystia/yorc/v4/tosca"
)

type delegateExecutor struct{}

func (de *delegateExecutor) ExecDelegate(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, delegateOperation string) error {
	log.Debugf("Entering plugin ExecDelegate")
	// Here is how to retrieve config parameters from Yorc config file
	if cfg.Infrastructures["heappe"] != nil {
		log.Printf("********Got myinfra infrastructure configured")
		for _, k := range cfg.Infrastructures["url"].Keys() {
			log.Printf("configuration key: %s", k)
		}
		log.Printf("*******Secret key: %q", cfg.Infrastructures["heappe"].GetStringOrDefault("url", "not found!"))

		// TODO: add here the code retrieving properties to connect to the API
		// allowing to allocated compute instances/connect to your
		// infrastructure
	}

	// Get a consul client to interact with the deployment API
	cc, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}
	kv := cc.KV()

	// Get node instances related to this task (may be a subset of all instances for a scaling operation for instance)
	instances, err := tasks.GetInstances(kv, taskID, deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Emit events and logs on instance status change
	for _, instanceName := range instances {
		deployments.SetInstanceStateWithContextualLogs(ctx, kv, deploymentID, nodeName, instanceName, tosca.NodeStateCreating)
	}

	// Use the deployments api to get info about the node to provision
	nodeType, err := deployments.GetNodeType(cc.KV(), deploymentID, nodeName)
	if err != nil {
		return err
	}

	// Emit a log or an event
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("**********Provisioning node %q of type %q", nodeName, nodeType)

	operation := strings.ToLower(delegateOperation)
	exec, err := newExecution(ctx, cfg, taskID, deploymentID, nodeName, operation)
	if err != nil {
		return err
	}

	err = exec.execute(ctx)
	if err != nil {
		return err
	}

	for _, instanceName := range instances {
		// TODO: add here the code allowing to create a Compute Instance
		deployments.SetInstanceStateWithContextualLogs(ctx, kv, deploymentID, nodeName, instanceName, tosca.NodeStateStarted)
	}
	return nil
}
