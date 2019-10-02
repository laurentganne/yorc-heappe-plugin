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

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/prov"
)

type execution interface {
	executeAsync(ctx context.Context) (*prov.Action, time.Duration, error)
	execute(ctx context.Context) error
}

type jobExecution struct {
	kv            *api.KV
	cfg           config.Configuration
	deploymentID  string
	taskID        string
	nodeName      string
	operationName string
	jobID         int64
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
	if !isJob {
		return nil, errors.Errorf("operation %q supported only for nodes derived from %q", operation, heappeJobType)
	}

	exec := &jobExecution{
		kv:            kv,
		cfg:           cfg,
		deploymentID:  deploymentID,
		taskID:        taskID,
		nodeName:      nodeName,
		operationName: operation,
	}

	return exec, err
}

func (exec *jobExecution) executeAsync(ctx context.Context) (*prov.Action, time.Duration, error) {
	return nil, 0, nil
}

func (exec *jobExecution) execute(ctx context.Context) error {

	var err error
	switch exec.operationName {
	case installOperation:
		err = exec.createJob(ctx)
	case uninstallOperation:
		err = exec.deleteJob(ctx)
	default:
		err = errors.Errorf("Unsupported operation %q", exec.operationName)
	}

	return err
}

func (exec *jobExecution) createJob(ctx context.Context) error {
	return nil
}

func (exec *jobExecution) deleteJob(ctx context.Context) error {
	return nil
}
