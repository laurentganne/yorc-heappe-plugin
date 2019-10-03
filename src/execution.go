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
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/prov"
)

const (
	infrastructureType      = "HEAppE"
	locationURLPropertyName = "URL"
	jobIdConsulAttribute    = "job_id"
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

func (e *jobExecution) executeAsync(ctx context.Context) (*prov.Action, time.Duration, error) {
	return nil, 0, nil
}

func (e *jobExecution) execute(ctx context.Context) error {

	var err error
	switch e.operationName {
	case installOperation:
		err = e.createJob(ctx)
	case uninstallOperation:
		err = e.deleteJob(ctx)
	default:
		err = errors.Errorf("Unsupported operation %q", e.operationName)
	}

	return err
}

func (e *jobExecution) createJob(ctx context.Context) error {

	jobSpec, err := e.getJobSpecification()
	if err != nil {
		return err
	}

	heappeClient, err := e.getHEAppEClient()
	if err != nil {
		return err
	}

	jobId, err := heappeClient.CreateJob(jobSpec)
	if err != nil {
		return err
	}

	// Store the job id
	err = deployments.SetAttributeForAllInstances(e.kv, e.deploymentID, e.nodeName,
		jobIdConsulAttribute, strconv.FormatInt(jobId, 10))
	if err != nil {
		err = errors.Wrapf(err, "Job %d created on HEAppE, but failed to store this job id", jobId)
	}
	return err
}

func (e *jobExecution) deleteJob(ctx context.Context) error {

	jobID, err := e.getJobID()
	if err != nil {
		return err
	}

	heappeClient, err := e.getHEAppEClient()
	if err != nil {
		return err
	}

	return heappeClient.DeleteJob(jobID)
}

func (e *jobExecution) getJobID() (int64, error) {
	var jobID int64

	val, err := deployments.GetInstanceAttributeValue(e.kv, e.deploymentID, e.nodeName, "0", "job_id")
	if err != nil {
		return jobID, err
	} else if val == nil {
		return jobID, errors.Errorf("Found no job id for deployment %s node %s", e.deploymentID, e.nodeName)
	}

	strVal := val.RawString()
	jobID, err = strconv.ParseInt(strVal, 10, 64)
	if err != nil {
		err = errors.Wrapf(err, "Unexpected Job ID value %q for deployment %s node %s", e.deploymentID, e.nodeName)
	}

	return jobID, err

}

func (e *jobExecution) getJobSpecification() (JobSpecification, error) {

	var jobSpec JobSpecification
	var err error

	jobSpec.Name, err = deployments.GetStringNodePropertyValue(e.kv, e.deploymentID,
		e.nodeName, jobSpecificationProperty, "name")
	if err != nil {
		return jobSpec, err
	}

	jobSpec.Project, err = deployments.GetStringNodePropertyValue(e.kv, e.deploymentID,
		e.nodeName, jobSpecificationProperty, "project")
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
		*(fieldPropName.field), err = getIntNodePropertyValue(e.kv, e.deploymentID,
			e.nodeName, jobSpecificationProperty, fieldPropName.propName)
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

func (e *jobExecution) getHEAppEClient() (HEAppEClient, error) {

	url := e.cfg.Infrastructures[infrastructureType].GetString(locationURLPropertyName)
	if url == "" {
		return nil, errors.Errorf("No URL defined in HEAppE location configuration")
	}

	username, err := deployments.GetStringNodePropertyValue(e.kv, e.deploymentID,
		e.nodeName, "user")
	if err != nil {
		return nil, err
	}
	if username == "" {
		return nil, errors.Errorf("No user defined in deployment %s node %s", e.deploymentID, e.nodeName)

	}

	password, err := deployments.GetStringNodePropertyValue(e.kv, e.deploymentID,
		e.nodeName, "password")
	if err != nil {
		return nil, err
	}

	return NewBasicAuthClient(url, username, password), err
}
