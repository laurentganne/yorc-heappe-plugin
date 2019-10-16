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
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/prov"
)

const (
	infrastructureType                    = "heappe"
	locationURLPropertyName               = "url"
	locationUserPropertyName              = "user"
	locationPasswordPropertyName          = "password"
	locationJobMonitoringTimeInterval     = "job_monitoring_time_interval"
	locationDefaultMonitoringTimeInterval = 5 * time.Second
	jobIDConsulAttribute                  = "job_id"
	heappeJobType                         = "org.heappe.nodes.Job"
	heappeSendDatasetType                 = "org.heappe.nodes.Dataset"
	heappeReceiveDatasetType              = "org.heappe.nodes.Results"
)

type execution interface {
	resolveExecution() error
	executeAsync(ctx context.Context) (*prov.Action, time.Duration, error)
	execute(ctx context.Context) error
}

func newExecution(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string,
	operation prov.Operation) (execution, error) {

	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return nil, err
	}
	kv := consulClient.KV()

	var exec execution
	isJob, err := deployments.IsNodeDerivedFrom(kv, deploymentID, nodeName, heappeJobType)
	if err != nil {
		return exec, err
	}

	if isJob {
		locationMgr, err := locations.GetManager(cfg)
		if err != nil {
			return nil, err
		}
		locationProps, err := locationMgr.GetLocationPropertiesForNode(deploymentID,
			nodeName, infrastructureType)
		if err != nil {
			return nil, err
		}

		monitoringTimeInterval := locationProps.GetDuration(locationJobMonitoringTimeInterval)
		if monitoringTimeInterval <= 0 {
			// Default value
			monitoringTimeInterval = locationDefaultMonitoringTimeInterval
		}
		exec = &jobExecution{
			kv:                     kv,
			cfg:                    cfg,
			deploymentID:           deploymentID,
			taskID:                 taskID,
			nodeName:               nodeName,
			operation:              operation,
			MonitoringTimeInterval: monitoringTimeInterval,
		}

		return exec, err
	} else {

		isReceiveDataset := false
		isSendDataset, err := deployments.IsNodeDerivedFrom(kv, deploymentID, nodeName, heappeSendDatasetType)
		if err != nil {
			return exec, errors.Wrapf(err, "Could not get type for deployment %s node %s", deploymentID, nodeName)
		}
		if !isSendDataset {
			isReceiveDataset, err = deployments.IsNodeDerivedFrom(kv, deploymentID, nodeName, heappeReceiveDatasetType)
			if err != nil {
				return exec, errors.Wrapf(err, "Could not get type for deployment %s node %s", deploymentID, nodeName)
			}
		}

		if !isSendDataset && !isReceiveDataset {
			return exec, errors.Errorf("operation %q supported only for nodes derived from %q, %q or %q",
				operation, heappeJobType, heappeSendDatasetType, heappeReceiveDatasetType)
		}

		exec = &datasetTransferExecution{
			kv:           kv,
			cfg:          cfg,
			deploymentID: deploymentID,
			taskID:       taskID,
			nodeName:     nodeName,
			operation:    operation,
		}
	}

	return exec, exec.resolveExecution()
}

func getHEAppEClient(cfg config.Configuration, deploymentID, nodeName string) (HEAppEClient, error) {

	locationMgr, err := locations.GetManager(cfg)
	if err != nil {
		return nil, err
	}

	locationProps, err := locationMgr.GetLocationPropertiesForNode(deploymentID,
		nodeName, infrastructureType)
	if err != nil {
		return nil, err
	}

	url := locationProps.GetString(locationURLPropertyName)
	if url == "" {
		return nil, errors.Errorf("No URL defined in HEAppE location configuration")
	}
	username := locationProps.GetString("user")
	if username == "" {
		return nil, errors.Errorf("No user defined in deployment %s node %s",
			deploymentID, nodeName)

	}
	password := locationProps.GetString("password")

	return NewBasicAuthClient(url, username, password), err
}
