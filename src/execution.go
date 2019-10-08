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
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/prov"
)

const (
	infrastructureType                    = "heappe"
	locationURLPropertyName               = "url"
	locationJobMonitoringTimeInterval     = "job_monitoring_time_interval"
	locationDefaultMonitoringTimeInterval = 5 * time.Second
	jobIDConsulAttribute                  = "job_id"
	heappeJobType                         = "org.heappe.nodes.Job"
	heappeDatasetTransferType             = "org.heappe.nodes.TransferDataset"
)

type execution interface {
	executeAsync(ctx context.Context) (*prov.Action, time.Duration, error)
	execute(ctx context.Context) error
}

func newExecution(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName, operation string) (execution, error) {
	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return nil, err
	}
	kv := consulClient.KV()

	isJob, err := deployments.IsNodeDerivedFrom(kv, deploymentID, nodeName, heappeJobType)
	if err != nil {
		return nil, err
	}

	if isJob {
		monitoringTimeInterval := cfg.Infrastructures[infrastructureType].GetDuration(locationJobMonitoringTimeInterval)
		if monitoringTimeInterval <= 0 {
			// Default value
			monitoringTimeInterval = locationDefaultMonitoringTimeInterval
		}
		exec := &jobExecution{
			kv:                     kv,
			cfg:                    cfg,
			deploymentID:           deploymentID,
			taskID:                 taskID,
			nodeName:               nodeName,
			operationName:          operation,
			MonitoringTimeInterval: monitoringTimeInterval,
		}

		return exec, err
	}

	isDatasetTransfer, err := deployments.IsNodeDerivedFrom(kv, deploymentID, nodeName, heappeDatasetTransferType)
	if !isDatasetTransfer {

		return nil, errors.Errorf("operation %q supported only for nodes derived from %q or %q",
			operation, heappeJobType, heappeDatasetTransferType)
	}

	exec := &datasetTransferExecution{
		kv:            kv,
		cfg:           cfg,
		deploymentID:  deploymentID,
		taskID:        taskID,
		nodeName:      nodeName,
		operationName: operation,
	}

	return exec, err
}

func getHEAppEClient(cfg config.Configuration, deploymentID, nodeName string) (HEAppEClient, error) {

	url := cfg.Infrastructures[infrastructureType].GetString(locationURLPropertyName)
	if url == "" {
		return nil, errors.Errorf("No URL defined in HEAppE location configuration")
	}
	kv := consulutil.GetKV()
	username, err := deployments.GetStringNodePropertyValue(kv, deploymentID,
		nodeName, "user")
	if err != nil {
		return nil, err
	}
	if username == "" {
		return nil, errors.Errorf("No user defined in deployment %s node %s", deploymentID, nodeName)

	}

	password, err := deployments.GetStringNodePropertyValue(kv, deploymentID,
		nodeName, "password")
	if err != nil {
		return nil, err
	}

	return NewBasicAuthClient(url, username, password), err
}
