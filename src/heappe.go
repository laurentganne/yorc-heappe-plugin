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
	"net/http"

	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/log"
)

const (
	heappeAuthREST          = "/heappe/UserAndLimitationManagement/AuthenticateUserPassword"
	heappeCreateJobREST     = "/heappe/JobManagement/CreateJob"
	heappeSubmitJobREST     = "/heappe/JobManagement/SubmitJob"
	heappeCancelJobREST     = "/heappe/JobManagement/CancelJob"
	heappeDeleteJobREST     = "/heappe/JobManagement/DeleteJob"
	heappeJobInfoREST       = "/heappe/JobManagement/GetCurrentInfoForJob"
	heappeJobStateQueued    = "QUEUED"
	heappeJobStateRunning   = "RUNNING"
	heappeJobStateCompleted = "COMPLETED"
	heappeJobStateFailed    = "FAILED"
	heappeJobStateCanceled  = "CANCELED"
)

// Client is the client interface to HEAppE service
type Client interface {
	CreateJob(job JobSpecification) (int64, error)
	SubmitJob(jobID int64) error
	CancelJob(jobID int64) error
	DeleteJob(jobID int64) error
	GetJobState(jobID int64) (string, error)
}

// NewBasicAuthClient returns a client performing a basic user/pasword authentication
func NewBasicAuthClient(url, username, password string) Client {
	return &heappeClient{
		auth: Authentication{
			Credentials: PasswordCredentials{
				Username: username,
				Password: password,
			},
		},
		httpClient: getHTTPClient(url),
	}
}

type heappeClient struct {
	auth       Authentication
	sessionID  string
	httpClient *httpclient
}

// CreateJob creates a HEAppE job
func (h *heappeClient) CreateJob(job JobSpecification) (int64, error) {

	// First authenticate
	var jobID int64
	var err error
	h.sessionID, err = h.authenticate()
	if err != nil {
		return jobID, err
	}

	params := JobCreateRESTParams{
		JobSpecification: job,
		SessionCode:      h.sessionID,
	}
	var jobResponse JobRESTResponse

	err = h.httpClient.doRequest(http.MethodPost, heappeCreateJobREST, http.StatusOK, params, &jobResponse)
	jobID = jobResponse.ID
	if err != nil {
		err = errors.Wrap(err, "Failed to create job")
	}

	return jobID, err
}

// SubmitJob submits a HEAppE job
func (h *heappeClient) SubmitJob(jobID int64) error {

	params := JobSubmitRESTParams{
		CreatedJobInfoID: jobID,
		SessionCode:      h.sessionID,
	}

	var jobResponse JobRESTResponse

	err := h.httpClient.doRequest(http.MethodPost, heappeSubmitJobREST, http.StatusOK, params, &jobResponse)
	if err != nil {
		err = errors.Wrap(err, "Failed to submit job")
	}

	return err
}

// CancelJob cancels a HEAppE job
func (h *heappeClient) CancelJob(jobID int64) error {

	params := JobInfoRESTParams{
		SubmittedJobInfoID: jobID,
		SessionCode:        h.sessionID,
	}

	var jobResponse JobRESTResponse

	err := h.httpClient.doRequest(http.MethodPost, heappeCancelJobREST, http.StatusOK, params, &jobResponse)
	if err != nil {
		err = errors.Wrap(err, "Failed to cancel job")
	}

	return err
}

// DeleteJob delete a HEAppE job
func (h *heappeClient) DeleteJob(jobID int64) error {

	params := JobInfoRESTParams{
		SubmittedJobInfoID: jobID,
		SessionCode:        h.sessionID,
	}

	var response string

	err := h.httpClient.doRequest(http.MethodPost, heappeDeleteJobREST, http.StatusOK, params, &response)
	if err != nil {
		err = errors.Wrap(err, "Failed to delete job")
	}

	return err
}

// GetJobState gets a HEAppE job state
func (h *heappeClient) GetJobState(jobID int64) (string, error) {

	params := JobInfoRESTParams{
		SubmittedJobInfoID: jobID,
		SessionCode:        h.sessionID,
	}

	var jobState string
	var jobResponse JobRESTResponse

	err := h.httpClient.doRequest(http.MethodGet, heappeJobInfoREST, http.StatusOK, params, &jobResponse)
	if err != nil {
		return jobState, errors.Wrap(err, "Failed to get job state")
	}

	stateValue := jobResponse.State
	switch stateValue {
	case 0, 1, 2:
		jobState = heappeJobStateQueued
	case 3:
		jobState = heappeJobStateRunning
	case 4:
		jobState = heappeJobStateCompleted
	case 5:
		jobState = heappeJobStateFailed
	case 6:
		jobState = heappeJobStateFailed // HEAppE state canceled
	default:
		log.Printf("Error getting state for job %d, unexpected state %d, considering it failed", jobID, stateValue)
		jobState = heappeJobStateFailed
	}

	return jobState, err
}

func (h *heappeClient) authenticate() (string, error) {
	var sessionID string

	err := h.httpClient.doRequest(http.MethodPost, heappeAuthREST, http.StatusOK, h.auth, &sessionID)
	if err != nil {
		return sessionID, errors.Wrap(err, "Failed to authenticate to HEAppE")
	}
	if len(sessionID) == 0 {
		err = errors.Errorf("Failed to open a HEAppE session")
	}

	return sessionID, err
}
