// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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
	"time"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tosca"
)

const (
	heappeJobType      = "org.heappe.nodes.Job"
	installOperation   = "install"
	uninstallOperation = "uninstall"
)

type operationExecutor struct{}

func (e *operationExecutor) ExecAsyncOperation(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation, stepName string) (*prov.Action, time.Duration, error) {

	consulClient, err := cfg.GetConsulClient()
	if err != nil {
		return nil, 0, err
	}
	kv := consulClient.KV()

	isJob, err := deployments.IsNodeDerivedFrom(kv, deploymentID, nodeName, heappeJobType)
	if err != nil {
		return nil, 0, err
	}
	if !isJob {
		return nil, 0, errors.Errorf("operation %q supported only for nodes derived from %q", operation.Name, heappeJobType)
	}

	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf(
		"************* Executing asynchronous operation %q step %q on node %q", operation.Name, stepName, nodeName)
	return nil, 0, nil
}

func (e *operationExecutor) ExecOperation(ctx context.Context, cfg config.Configuration, taskID, deploymentID, nodeName string, operation prov.Operation) error {

	// Printing Yorc logs at different levels in the plugin,
	// Yorc server will filter these logs according to its logging level

	// Printing a debug level message
	log.Debugf("Entering ExecOperation")
	// Printing an INFO level message
	log.Printf("Executing operation %q", operation.Name)

	var delegateOperation string
	operationName := strings.ToLower(operation.Name)
	switch operationName {
	case "standard.create":
		delegateOperation = "install"
	case "standard.delete":
		delegateOperation = "uninstall"
	case tosca.RunnableSubmitOperationName:
		log.Printf("***** ExecOperation will call job submit")
	case tosca.RunnableCancelOperationName:
		log.Printf("***** ExecOperation will call job cancel")
	default:
		return errors.Errorf("Unsupported operation %q", operation.Name)
	}

	_, err := cfg.GetConsulClient()
	if err != nil {
		return err
	}

	if cfg.Infrastructures["my-plugin"] != nil {
		for _, k := range cfg.Infrastructures["my-plugin"].Keys() {
			log.Printf("configuration key: %s", k)
		}
		log.Printf("Secret key: %q", cfg.Infrastructures["my-plugin"].GetStringOrDefault("test", "not found!"))
	}

	// Emit a log or an event
	events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).Registerf("******Executing operation %q on node %q", operation.Name, nodeName)

	if delegateOperation != "" {
		return new(delegateExecutor).ExecDelegate(ctx, cfg, taskID, deploymentID, nodeName, delegateOperation)
	}

	return nil
}
