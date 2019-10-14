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
	"os"
	"path/filepath"
	"strconv"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/deployments"
	"github.com/ystia/yorc/v4/events"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
	"github.com/ystia/yorc/v4/prov/operations"
	"golang.org/x/crypto/ssh"
)

const (
	jobIDEnvVar = "JOB_ID"
)

type datasetTransferExecution struct {
	kv             *api.KV
	cfg            config.Configuration
	deploymentID   string
	taskID         string
	nodeName       string
	operation      prov.Operation
	overlayPath    string
	artifacts      map[string]string
	envInputs      []*operations.EnvInput
	varInputsNames []string
}

func (e *datasetTransferExecution) executeAsync(ctx context.Context) (*prov.Action, time.Duration, error) {

	return nil, 0, errors.Errorf("Unsupported asynchronous operation %q for dataset transfer", e.operation.Name)

}

func (e *datasetTransferExecution) execute(ctx context.Context) error {

	var err error
	switch e.operation.Name {
	case installOperation, "standard.create":
		events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(
			"Starting dataset transfer %q", e.nodeName)
		err = e.transferDataset(ctx)
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.LogLevelINFO, e.deploymentID).Registerf(
				"Dataset transfer %q failed, error %s", e.nodeName, err.Error())

		}
	case uninstallOperation, "standard.delete":
		// Nothing to do
	default:
		err = errors.Errorf("Unsupported operation %q on dataset transfer", e.operation.Name)
	}

	return err
}

func (e *datasetTransferExecution) transferDataset(ctx context.Context) error {

	// Get files to transfer
	fileNames, err := e.getDatasetFileNames()
	if err != nil {
		return err
	}

	// Get details on destination where to transfer files
	heappeClient, err := getHEAppEClient(e.cfg, e.deploymentID, e.nodeName)
	if err != nil {
		return err
	}

	jobIDStr := e.getValueFromEnvInputs(jobIDEnvVar)
	if jobIDStr == "" {
		return errors.Errorf("Failed to get associated job ID")
	}
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		err = errors.Wrapf(err, "Unexpected Job ID value %q for deployment %s node %s",
			jobIDStr, e.deploymentID, e.nodeName)
		return err
	}

	transferMethod, err := heappeClient.GetFileTransferMethod(jobID)
	if err != nil {
		return err
	}

	defer func() {
		heappeClient.EndFileTransfer(jobID, transferMethod)
	}()

	clientConfig, err := getSSHClientConfig(transferMethod.Credentials.Username,
		transferMethod.Credentials.PrivateKey)
	if err != nil {
		return errors.Wrapf(err, "Failed to create a SSH client using HEAppE file transfer credentials")
	}

	client := scp.NewClient(transferMethod.ServerHostname+":22", &clientConfig)

	// Connect to the remote server
	err = client.Connect()
	if err != nil {
		// TODO: uncomment this error when it will work
		// return errors.Wrapf(err, "Failed to connect to remote server using HEAppE file transfer credentials")
		log.Printf("!!!! ERROR connection to transfer files failed: %+s", err.Error())
	}
	defer client.Close()

	// Transfer each file in the dataset
	for _, filename := range fileNames {

		f, err := os.Open(filename)
		if err != nil {
			return errors.Wrapf(err, "Failed to open dataset file %s", filename)
		}
		defer f.Close()

		// Finaly, copy the file over
		// Usage: CopyFile(fileReader, remotePath, permission)

		err = client.CopyFile(f, filepath.Join(transferMethod.SharedBasepath, filename), "0744")
		if err != nil {
			return errors.Wrapf(err, "Failed to copy dataset file %s to remote server", filename)
		}
	}

	return err
}

func (e *datasetTransferExecution) getDatasetFileNames() ([]string, error) {

	var fileNames []string

	datasetFileName := e.artifacts["dataset"]
	if datasetFileName == "" {
		return fileNames, errors.Errorf("No dataset provided")
	}

	// TODO: manage the case where this dataset needs to be unzipped
	fileNames = []string{datasetFileName}

	return fileNames, nil
}

func (e *datasetTransferExecution) resolveExecution() error {
	log.Debugf("Preparing execution of operation %q on node %q for deployment %q", e.operation.Name, e.nodeName, e.deploymentID)
	ovPath, err := operations.GetOverlayPath(e.kv, e.cfg, e.taskID, e.deploymentID)
	if err != nil {
		return err
	}
	e.overlayPath = ovPath

	if err = e.resolveInputs(); err != nil {
		return err
	}
	if err = e.resolveArtifacts(); err != nil {
		return err
	}

	return err
}

func (e *datasetTransferExecution) resolveInputs() error {
	var err error
	// TODO switch to debug mode
	log.Debugf("Get environment inputs for node:%q", e.nodeName)
	e.envInputs, e.varInputsNames, err = operations.ResolveInputsWithInstances(e.kv, e.deploymentID, e.nodeName, e.taskID, e.operation, nil, nil)
	// TODO switch to debug mode
	log.Printf("Environment inputs: %v", e.envInputs)
	return err
}

func (e *datasetTransferExecution) resolveArtifacts() error {
	var err error
	log.Debugf("Get artifacts for node:%q", e.nodeName)
	e.artifacts, err = deployments.GetArtifactsForNode(e.kv, e.deploymentID, e.nodeName)
	log.Debugf("Resolved artifacts: %v", e.artifacts)
	return err
}

func getSSHClientConfig(username, privateKey string) (ssh.ClientConfig, error) {

	var clientConfig ssh.ClientConfig

	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err == nil {
		clientConfig = ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
	}

	return clientConfig, err
}

func (e *datasetTransferExecution) getValueFromEnvInputs(envVar string) string {

	var result string
	for _, envInput := range e.envInputs {
		if envInput.Name == envVar {
			result = envInput.Value
			break
		}
	}
	return result

}
