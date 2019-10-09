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
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/prov"
)

type datasetTransferExecution struct {
	kv            *api.KV
	cfg           config.Configuration
	deploymentID  string
	taskID        string
	nodeName      string
	operationName string
}

func (e *datasetTransferExecution) executeAsync(ctx context.Context) (*prov.Action, time.Duration, error) {

	return nil, 0, errors.Errorf("Unsupported asynchronous operation %q for dataset transfer", e.operationName)

}

func (e *datasetTransferExecution) execute(ctx context.Context) error {

	var err error
	switch e.operationName {
	case installOperation:
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(
			"Starting dataset transfer %q", e.nodeName)
		err = e.transferDataset(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(
				"Dataset transfer %q failed, error %s", e.nodeName, err.Error())

		}
	case uninstallOperation:
		// Nothing to do
	default:
		err = errors.Errorf("Unsupported operation %q on dataset transfer", e.operationName)
	}

	return err
}

func (e *datasetTransferExecution) transferDataset(ctx context.Context) error {
	return errors.Errorf("TransferDataset not yet implemented")
}
