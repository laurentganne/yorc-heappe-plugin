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
	"strings"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/helper/consulutil"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/scheduling"
)

const (
	actionDataSessionID       = "sessionID"
	jobStatePending           = "PENDING"
	jobStateRunning           = "RUNNING"
	jobStateCompleted         = "COMPLETED"
	jobStateFailed            = "FAILED"
	jobStateCanceled          = "CANCELED"
	actionDataOffsetKeyFormat = "%d_%d_%d"
)

type fileType int

const (
	logFile fileType = iota
	progressFile
	standardErrorFile
	standardOutputFile
)

var fileTypes = []fileType{logFile, progressFile, standardErrorFile, standardOutputFile}

type actionOperator struct {
}

type actionData struct {
	jobID     int64
	taskID    string
	nodeName  string
	sessionID string
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

	// Set session ID if defined, else a new session will be created
	actionData.sessionID, ok = action.Data[actionDataSessionID]
	if ok && actionData.sessionID != "" {
		heappeClient.SetSessionID(actionData.sessionID)
	}

	jobInfo, err := heappeClient.GetJobInfo(actionData.jobID)
	if err != nil {
		return true, err
	}

	if actionData.sessionID == "" {
		// Storing the session ID for next job state check
		err = scheduling.UpdateActionData(nil, action.ID, actionDataSessionID, heappeClient.GetSessionID())
		if err != nil {
			return true, errors.Wrapf(err, "failed to update action data for deployment %s node %s", deploymentID, actionData.nodeName)
		}
	}

	jobState := getJobState(jobInfo)
	previousJobState, err := deployments.GetInstanceStateString(consulutil.GetKV(), deploymentID, actionData.nodeName, "0")
	if err != nil {
		return true, errors.Wrapf(err, "failed to get instance state for job %d", actionData.jobID)
	}
	if previousJobState != jobState {
		deployments.SetInstanceStateStringWithContextualLogs(ctx, consulutil.GetKV(), deploymentID, actionData.nodeName, "0", jobState)
	}

	// Log job outputs
	err = o.getJobOutputs(ctx, heappeClient, deploymentID, actionData.nodeName, action, jobInfo)
	if err != nil {
		log.Printf("Failed to get job outputs : %s", err.Error())
	}

	// See if monitoring must be continued and set job state if terminated
	switch jobState {
	case jobStateCompleted:
		// job has been done successfully : unregister monitoring
		deregister = true
	case jobStatePending, jobStateRunning:
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

func (o *actionOperator) getJobOutputs(ctx context.Context, heappeClient HEAppEClient, deploymentID, nodeName string,
	action *prov.Action, jobInfo SubmittedJobInfo) error {

	var err error
	var offsets []TaskFileOffset
	for _, task := range jobInfo.Tasks {
		for _, fType := range fileTypes {
			var tOffset TaskFileOffset
			tOffset.SubmittedTaskInfoID = task.ID
			tOffset.FileType = int(fType)
			tOffset.Offset, err = getOffset(jobInfo.ID, task.ID, tOffset.FileType, action)
			if err != nil {
				return errors.Wrapf(err, "Failed to compute offset for log file on deployment %s node %s job %d ",
					deploymentID, nodeName, jobInfo.ID)
			}

			offsets = append(offsets, tOffset)
		}
	}

	contents, err := heappeClient.DownloadPartsOfJobFilesFromCluster(jobInfo.ID, offsets)
	if err != nil {
		return err
	}

	// Print contents
	for _, fileContent := range contents {
		if strings.TrimSpace(fileContent.Content) != "" {
			fileTypeStr := displayFileType(fileContent.FileType)
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString(
				fmt.Sprintf("Job %d task %d %s:", jobInfo.ID, fileContent.SubmittedTaskInfoID, fileTypeStr))
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, deploymentID).RegisterAsString("\n" + fileContent.Content)

			// Save the new offset
			offsetKey := getActionDataOffsetKey(jobInfo.ID, fileContent.SubmittedTaskInfoID, fileContent.FileType)
			err = scheduling.UpdateActionData(nil, action.ID, offsetKey, strconv.FormatInt(fileContent.Offset, 10))
			if err != nil {
				return errors.Wrapf(err, "failed to update action data for deployment %s node %s job %d task %d %s",
					deploymentID, nodeName, jobInfo.ID, fileContent.SubmittedTaskInfoID, fileTypeStr)
			}
		}
	}

	return err
}

func getOffset(jobID, taskID int64, fileType int, action *prov.Action) (int64, error) {

	offsetKey := getActionDataOffsetKey(jobID, taskID, fileType)
	offsetStr := action.Data[offsetKey]
	var err error
	var offset int64
	if offsetStr != "" {
		offset, err = strconv.ParseInt(offsetStr, 10, 64)
	}
	return offset, err
}

func getActionDataOffsetKey(jobID, taskID int64, fileType int) string {
	return fmt.Sprintf(actionDataOffsetKeyFormat, jobID, taskID, fileType)
}

func getJobState(jobInfo SubmittedJobInfo) string {
	var strValue string
	switch jobInfo.State {
	case 0, 1, 2:
		strValue = jobStatePending
	case 3:
		strValue = jobStateRunning
	case 4:
		strValue = jobStateCompleted
	case 5:
		strValue = jobStateFailed
	case 6:
		strValue = jobStateFailed // HEAppE state canceled
	default:
		log.Printf("Error getting state for job %d, unexpected state %d, considering it failed", jobInfo.ID, strValue)
		strValue = jobStateFailed
	}
	return strValue
}

func displayFileType(fType int) string {
	var strValue string

	switch fileType(fType) {
	case logFile:
		strValue = "Log file"
	case progressFile:
		strValue = "Progress file"
	case standardErrorFile:
		strValue = "Standard Error"
	case standardOutputFile:
		strValue = "Standard Output"
	default:
		log.Printf("Unknown file type %d, unexpected state %d", fType)
		strValue = "Unknwown file"
	}
	return strValue
}
