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
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
)

type actionOperator struct {
}

type actionData struct {
	jobID    int64
	taskID   string
	nodeName string
}

func (o *actionOperator) ExecAction(ctx context.Context, cfg config.Configuration, taskID, deploymentID string, action *prov.Action) (bool, error) {
	log.Debugf("Execute Action with ID:%q, taskID:%q, deploymentID:%q", action.ID, taskID, deploymentID)

	if action.ActionType == "heappe-job-monitoring" {
		deregister, err := o.monitorJob(ctx, cfg, deploymentID, action)
		if err != nil {
			// action scheduling needs to be unregistered
			return true, err
		}

		return deregister, nil
	}
	return true, errors.Errorf("Unsupported actionType %q", action.ActionType)
}

func (o *actionOperator) monitorJob(ctx context.Context, cfg config.Configuration, deploymentID string, action *prov.Action) (bool, error) {
	var (
		err        error
		deregister bool
		ok         bool
	)

	actionData := &actionData{}
	// Check nodeName
	actionData.nodeName, ok = action.Data["nodeName"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information nodeName for actionType:%q", action.ActionType)
	}
	// Check jobID
	jobIDstr, ok := action.Data["jobID"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information jobID for actionType:%q", action.ActionType)
	}
	actionData.jobID, err = strconv.ParseInt(jobIDstr, 10, 64)
	if err != nil {
		return true, errors.Wrapf(err, "Unexpected Job ID value %q for deployment %s node %s", jobIDstr, deploymentID, actionData.nodeName)
	}

	// Check taskID
	actionData.taskID, ok = action.Data["taskID"]
	if !ok {
		return true, errors.Errorf("Missing mandatory information taskID for actionType:%q", action.ActionType)
	}

	heappeClient, err := getHEAppEClient(cfg, deploymentID, actionData.nodeName)
	if err != nil {
		return true, err
	}
	jobState, err := heappeClient.GetJobState(actionData.jobID)
	if err != nil {
		return true, err
	}

	previousJobState, err := deployments.GetInstanceStateString(consulutil.GetKV(), deploymentID, actionData.nodeName, "0")
	if err != nil {
		return true, errors.Wrapf(err, "failed to get instance state for job %d", actionData.jobID)
	}
	if previousJobState != jobState {
		deployments.SetInstanceStateStringWithContextualLogs(ctx, consulutil.GetKV(), deploymentID, actionData.nodeName, "0", jobState)
	}

	// See if monitoring must be continued and set job state if terminated
	switch jobState {
	case heappeJobStateCompleted:
		// job has been done successfully : unregister monitoring
		deregister = true
	case heappeJobStatePending, heappeJobStateRunning:
		// job's still running or its state is about to be set definitively: monitoring is keeping on (deregister stays false)
	default:
		// Other cases as FAILED, CANCELED : error is return with job state and job info is logged
		deregister = true
		// Log event containing all the slurm information

		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelERROR, deploymentID).RegisterAsString(fmt.Sprintf("job state:%+v", jobState))
		// Error to be returned
		err = errors.Errorf("job with ID: %d finished unsuccessfully with state: %q", actionData.jobID, jobState)
	}

	return deregister, err
}
