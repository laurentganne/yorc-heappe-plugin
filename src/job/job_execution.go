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

package job

import (
	"context"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/laurentganne/yorc-heappe-plugin/v1/heappe"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/tosca"
)

const (
	installOperation                      = "install"
	uninstallOperation                    = "uninstall"
	jobSpecificationProperty              = "jobSpecification"
	infrastructureType                    = "heappe"
	locationJobMonitoringTimeInterval     = "job_monitoring_time_interval"
	locationDefaultMonitoringTimeInterval = 5 * time.Second
	jobIDConsulAttribute                  = "job_id"
)

// JobExecution holds job properties
type JobExecution struct {
	KV                     *api.KV
	Cfg                    config.Configuration
	DeploymentID           string
	TaskID                 string
	NodeName               string
	Operation              prov.Operation
	JobID                  int64
	MonitoringTimeInterval time.Duration
}

func (e *JobExecution) ExecuteAsync(ctx context.Context) (*prov.Action, time.Duration, error) {
	if e.Operation.Name != tosca.RunnableRunOperationName {
		return nil, 0, errors.Errorf("Unsupported asynchronous operation %q", e.Operation.Name)
	}

	jobID, err := e.getJobID()
	if err != nil {
		return nil, 0, err
	}

	data := make(map[string]string)
	data["taskID"] = e.TaskID
	data["nodeName"] = e.NodeName
	data["jobID"] = strconv.FormatInt(jobID, 10)

	// TODO: use a configurable time duration
	return &prov.Action{ActionType: "heappe-job-monitoring", Data: data}, e.MonitoringTimeInterval, err
}

func (e *JobExecution) Execute(ctx context.Context) error {

	var err error
	switch e.Operation.Name {
	case installOperation, "standard.create":
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
			"Creating Job %q", e.NodeName)
		err = e.createJob(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
				"Failed to create Job %q, error %s", e.NodeName, err.Error())

		}
	case uninstallOperation, "standard.delete":
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
			"Deleting Job %q", e.NodeName)
		err = e.deleteJob(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
				"Failed to delete Job %q, error %s", e.NodeName, err.Error())

		}
	case tosca.RunnableSubmitOperationName:
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
			"Submitting Job %q", e.NodeName)
		err = e.submitJob(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
				"Failed to submit Job %q, error %s", e.NodeName, err.Error())

		}
	case tosca.RunnableCancelOperationName:
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
			"Canceling Job %q", e.NodeName)
		err = e.cancelJob(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.DeploymentID).Registerf(
				"Failed to cancel Job %q, error %s", e.NodeName, err.Error())

		}
	default:
		err = errors.Errorf("Unsupported operation %q", e.Operation.Name)
	}

	return err
}

func (e *JobExecution) ResolveExecution() error {
	return nil
}

func (e *JobExecution) createJob(ctx context.Context) error {

	jobSpec, err := e.getJobSpecification()
	if err != nil {
		return err
	}

	heappeClient, err := getHEAppEClient(e.Cfg, e.DeploymentID, e.NodeName)
	if err != nil {
		return err
	}

	jobID, err := heappeClient.CreateJob(jobSpec)
	if err != nil {
		return err
	}

	// Store the job id
	err = deployments.SetAttributeForAllInstances(e.KV, e.DeploymentID, e.NodeName,
		jobIDConsulAttribute, strconv.FormatInt(jobID, 10))
	if err != nil {
		err = errors.Wrapf(err, "Job %d created on HEAppE, but failed to store this job id", jobID)
	}
	return err
}

func (e *JobExecution) deleteJob(ctx context.Context) error {

	jobID, err := e.getJobID()
	if err != nil {
		return err
	}

	heappeClient, err := getHEAppEClient(e.Cfg, e.DeploymentID, e.NodeName)
	if err != nil {
		return err
	}

	return heappeClient.DeleteJob(jobID)
}

func (e *JobExecution) submitJob(ctx context.Context) error {

	jobID, err := e.getJobID()
	if err != nil {
		return err
	}

	heappeClient, err := getHEAppEClient(e.Cfg, e.DeploymentID, e.NodeName)
	if err != nil {
		return err
	}

	return heappeClient.SubmitJob(jobID)
}

func (e *JobExecution) cancelJob(ctx context.Context) error {

	jobID, err := e.getJobID()
	if err != nil {
		return err
	}

	heappeClient, err := getHEAppEClient(e.Cfg, e.DeploymentID, e.NodeName)
	if err != nil {
		return err
	}

	return heappeClient.CancelJob(jobID)
}

func (e *JobExecution) getJobID() (int64, error) {
	var jobID int64

	val, err := deployments.GetInstanceAttributeValue(e.KV, e.DeploymentID, e.NodeName, "0", jobIDConsulAttribute)
	if err != nil {
		return jobID, errors.Wrapf(err, "Failed to get job id for deployment %s node %s", e.DeploymentID, e.NodeName)
	} else if val == nil {
		return jobID, errors.Errorf("Found no job id for deployment %s node %s", e.DeploymentID, e.NodeName)
	}

	strVal := val.RawString()
	jobID, err = strconv.ParseInt(strVal, 10, 64)
	if err != nil {
		err = errors.Wrapf(err, "Unexpected Job ID value %q for deployment %s node %s", strVal, e.DeploymentID, e.NodeName)
	}

	return jobID, err

}

func (e *JobExecution) getJobSpecification() (heappe.JobSpecification, error) {

	var jobSpec heappe.JobSpecification
	var err error

	jobSpec.Name, err = deployments.GetStringNodePropertyValue(e.KV, e.DeploymentID,
		e.NodeName, jobSpecificationProperty, "name")
	if err != nil {
		return jobSpec, err
	}

	jobSpec.Project, err = deployments.GetStringNodePropertyValue(e.KV, e.DeploymentID,
		e.NodeName, jobSpecificationProperty, "project")
	if err != nil {
		return jobSpec, err
	}

	var fieldPropNames = []struct {
		field    *int
		propName string
	}{
		{field: &(jobSpec.ClusterNodeTypeID), propName: "clusterNodeTypeId"},
		{field: &(jobSpec.Priority), propName: "priority"},
		{field: &(jobSpec.MinCores), propName: "minCores"},
		{field: &(jobSpec.MaxCores), propName: "maxCores"},
		{field: &(jobSpec.WaitingLimit), propName: "waitingLimit"},
		{field: &(jobSpec.WalltimeLimit), propName: "walltimeLimit"},
	}

	for _, fieldPropName := range fieldPropNames {
		*(fieldPropName.field), err = getIntNodePropertyValue(e.KV, e.DeploymentID,
			e.NodeName, jobSpecificationProperty, fieldPropName.propName)
	}

	// Getting associated tasks
	tasks, err := deployments.GetNodePropertyValue(e.KV, e.DeploymentID, e.NodeName, jobSpecificationProperty, "tasks")
	if err != nil {
		return jobSpec, err
	}

	if tasks != nil && tasks.RawString() != "" {
		var ok bool
		jobSpec.Tasks = make([]heappe.TaskSpecification, 0)
		taskArray, ok := tasks.Value.([]interface{})
		if !ok {
			return jobSpec, errors.Errorf(
				"failed to retrieve job tasks specification for deployment %s node %s, wrong type for %+s",
				e.DeploymentID, e.NodeName, tasks.RawString())
		}

		for _, taskVal := range taskArray {
			attrMap, ok := taskVal.(map[string]interface{})
			if !ok {
				return jobSpec, errors.Errorf(
					"failed to retrieve task specification for deployment %s node %s, wrong type for %+v",
					e.DeploymentID, e.NodeName, taskVal)
			}

			var task heappe.TaskSpecification

			// Get string properties

			var taskPropConsulPropString = []struct {
				taskProp   *string
				consulProp string
			}{
				{taskProp: &(task.Name), consulProp: "name"},
				{taskProp: &(task.StandardOutputFile), consulProp: "standardOutputFile"},
				{taskProp: &(task.StandardErrorFile), consulProp: "standardErrorFile"},
				{taskProp: &(task.ProgressFile), consulProp: "progressFile"},
				{taskProp: &(task.LogFile), consulProp: "logFile"},
			}

			for _, props := range taskPropConsulPropString {
				rawValue, ok := attrMap[props.consulProp]
				if ok {
					val, ok := rawValue.(string)
					if !ok {
						return jobSpec, errors.Errorf(
							"Expected a string for deployment %s node %s task %s property %s, got %+v",
							e.DeploymentID, e.NodeName, task.Name, props.consulProp, rawValue)
					}
					*props.taskProp = val
				}
			}

			// Get int properties

			var taskPropConsulPropInt = []struct {
				taskProp   *int
				consulProp string
			}{
				{taskProp: &(task.CommandTemplateID), consulProp: "commandTemplateId"},
				{taskProp: &(task.MinCores), consulProp: "minCores"},
				{taskProp: &(task.MaxCores), consulProp: "maxCores"},
				{taskProp: &(task.WalltimeLimit), consulProp: "walltimeLimit"},
			}

			for _, props := range taskPropConsulPropInt {
				rawValue, ok := attrMap[props.consulProp]
				if ok {
					val, ok := rawValue.(string)
					if !ok {
						return jobSpec, errors.Errorf(
							"Expected an int for deployment %s node %s task %s property %s, got %+v",
							e.DeploymentID, e.NodeName, task.Name, props.consulProp, rawValue)
					}
					*props.taskProp, err = strconv.Atoi(val)
					if err != nil {
						return jobSpec, errors.Wrapf(err,
							"Cannot convert as an int value %q for deployment %s node %s task %s property %s",
							val, e.DeploymentID, e.NodeName, task.Name, props.consulProp)
					}
				}
			}

			// Get template parameters
			parameters, ok := attrMap["templateParameterValues"]
			if ok {
				paramsArray, ok := parameters.([]interface{})
				if !ok {
					return jobSpec, errors.Errorf(
						"failed to retrieve command template parameters for deployment %s node %s task %s, wrong type for %+v",
						e.DeploymentID, e.NodeName, task.Name, parameters)
				}

				for _, paramsVal := range paramsArray {
					attrMap, ok := paramsVal.(map[string]interface{})
					if !ok {
						return jobSpec, errors.Errorf(
							"failed to retrieve parameters for deployment %s node %s task %s, wrong type for %+v",
							e.DeploymentID, e.NodeName, task.Name, paramsVal)
					}

					var param heappe.CommandTemplateParameterValue
					v, ok := attrMap["commandParameterIdentifier"]
					if !ok {
						return jobSpec, errors.Errorf(
							"Failed to get command parameter identifier for deployment %s node %s task %s, parameter %+v",
							e.DeploymentID, e.NodeName, task.Name, attrMap)
					}
					param.CommandParameterIdentifier, ok = v.(string)
					if !ok {
						return jobSpec, errors.Errorf(
							"Failed to get command parameter identifier string value for deployment %s node %s task %s, identifier %+v",
							e.DeploymentID, e.NodeName, task.Name, v)
					}

					v, ok = attrMap["parameterValue"]
					if ok {
						param.ParameterValue, ok = v.(string)
						if !ok {
							return jobSpec, errors.Errorf(
								"Failed to get command parameter string value for deployment %s node %s task %s identifier %s, value %+v",
								e.DeploymentID, e.NodeName, task.Name, param.CommandParameterIdentifier, v)
						}

					}

					task.TemplateParameterValues = append(task.TemplateParameterValues, param)
				}
			}

			// Get environment variables
			parameters, ok = attrMap["environmentVariables"]
			if ok {
				paramsArray, ok := parameters.([]interface{})
				if !ok {
					return jobSpec, errors.Errorf(
						"failed to retrieve environment variables for deployment %s node %s task %s, wrong type for %+v",
						e.DeploymentID, e.NodeName, task.Name, parameters)
				}

				for _, paramsVal := range paramsArray {
					attrMap, ok := paramsVal.(map[string]interface{})
					if !ok {
						return jobSpec, errors.Errorf(
							"failed to retrieve environment variable for deployment %s node %s task %s, wrong type for %+v",
							e.DeploymentID, e.NodeName, task.Name, paramsVal)
					}

					var param heappe.EnvironmentVariable
					v, ok := attrMap["name"]
					if !ok {
						return jobSpec, errors.Errorf(
							"Failed to get environment variable name for deployment %s node %s task %s, parameter %+v",
							e.DeploymentID, e.NodeName, task.Name, attrMap)
					}
					param.Name, ok = v.(string)
					if !ok {
						return jobSpec, errors.Errorf(
							"Failed to get environment variable name string value for deployment %s node %s task %s, identifier %+v",
							e.DeploymentID, e.NodeName, task.Name, v)
					}

					v, ok = attrMap["value"]
					if ok {
						param.Value, ok = v.(string)
						if !ok {
							return jobSpec, errors.Errorf(
								"Failed to get environment variable string value for deployment %s node %s task %s identifier %s, value %+v",
								e.DeploymentID, e.NodeName, task.Name, param.Name, v)
						}

					}

					task.EnvironmentVariables = append(task.EnvironmentVariables, param)
				}
			}

			jobSpec.Tasks = append(jobSpec.Tasks, task)

		}
	}
	return jobSpec, err
}

func getIntNodePropertyValue(kv *api.KV, deploymentID, nodeName, propertyName string,
	nestedKeys ...string) (int, error) {

	var result int
	strVal, err := deployments.GetStringNodePropertyValue(kv, deploymentID, nodeName, propertyName, nestedKeys...)
	if err != nil {
		return result, err
	}

	if len(strVal) > 0 {
		result, err = strconv.Atoi(strVal)
	}
	return result, err
}

func getHEAppEClient(cfg config.Configuration, deploymentID, nodeName string) (heappe.HEAppEClient, error) {
	locationMgr, err := locations.GetManager(cfg)
	if err != nil {
		return nil, err
	}

	locationProps, err := locationMgr.GetLocationPropertiesForNode(deploymentID,
		nodeName, infrastructureType)
	if err != nil {
		return nil, err
	}

	return heappe.GetClient(locationProps)
}