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
)

const (
	heappeAuthREST      = "/heappe/UserAndLimitationManagement/AuthenticateUserPassword"
	heappeCreateJobREST = "/heappe/JobManagement/CreateJob"
)

// Client is the client interface to HEAppE service
type Client interface {
	CreateJob(job JobSpecification) (int64, error)
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
	var jobCreated JobCreateRESTResponse

	err = h.httpClient.doRequest(http.MethodPost, heappeCreateJobREST, http.StatusOK, params, &jobCreated)
	jobID = jobCreated.ID
	if err != nil {
		err = errors.Wrap(err, "Failed to create job")
	}

	return jobID, err
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
