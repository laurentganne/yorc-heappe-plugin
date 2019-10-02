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

// PasswordCredentials holds user/password to perform a basic authentication
type PasswordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Authentication parameters
type Authentication struct {
	Credentials PasswordCredentials `json:"credentials"`
}

// CommandTemplateParameterValue holds a command template parameter
type CommandTemplateParameterValue struct {
	CommandParameterIdentifier string `json:"commandParameterIdentifier"`
	ParameterValue             string `json:"parameterValue"`
}

// EnvironmentVariable holds an environment variable definition
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// TaskSpecification holds task properties
type TaskSpecification struct {
	Name                    string                          `json:"name"`
	CommandTemplateID       int                             `json:"commandTemplateID"`
	TemplateParameterValues []CommandTemplateParameterValue `json:"templateParameterValues,omitempty"`
	MinCores                int                             `json:"minCores"`
	MaxCores                int                             `json:"maxCores"`
	WalltimeLimit           int                             `json:"walltimeLimit"`
	StandardOutputFile      string                          `json:"standardOutputFile"`
	StandardErrorFile       string                          `json:"standardErrorFile"`
	ProgressFile            string                          `json:"progressFile"`
	LogFile                 string                          `json:"logFile"`
	EnvironmentVariables    []EnvironmentVariable           `json:"environmentVariables,omitempty"`
}

// JobSpecification holds job properties
type JobSpecification struct {
	Name              string              `json:"name"`
	Project           string              `json:"project"`
	ClusterNodeTypeID int                 `json:"clusterNodeTypeId"`
	Tasks             []TaskSpecification `json:"tasks"`
	Priority          int                 `json:"priority"`
	MinCores          int                 `json:"minCores"`
	MaxCores          int                 `json:"maxCores"`
	WaitingLimit      int                 `json:"waitingLimit"`
	WalltimeLimit     int                 `json:"walltimeLimit"`
}

// JobCreateRESTParams holds HEAppE REST API job creation parameters
type JobCreateRESTParams struct {
	JobSpecification JobSpecification `json:"jobSpecification"`
	SessionCode      string           `json:"sessionCode"`
}

// TemplateParameterRESTResponse holds template parameters description in a job
type TemplateParameterRESTResponse struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
}

// CommandTemplateRESTResponse holds a command template description in a job
type CommandTemplateRESTResponse struct {
	ID                 int64                           `json:"id"`
	Name               string                          `json:"name"`
	Description        string                          `json:"description"`
	Code               string                          `json:"code"`
	TemplateParameters []TemplateParameterRESTResponse `json:"templateParameters"`
}

// ClusterNodeTypeRESTResponse holds a node description in a job
type ClusterNodeTypeRESTResponse struct {
	ID               int64                         `json:"id"`
	Name             string                        `json:"name"`
	Description      string                        `json:"description"`
	NumberOfNodes    int                           `json:"numberOfNodes"`
	CoresPerNode     int                           `json:"coresPerNode"`
	MaxWalltime      int                           `json:"maxWalltime"`
	CommandTemplates []CommandTemplateRESTResponse `json:"commandTemplates"`
}

// TaskRESTResponse holds a task description in a job
type TaskRESTResponse struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	State            int     `json:"state"`
	AllocatedTime    float64 `json:"allocatedTime,omitempty"`
	AllocatedCoreIds string  `json:"allocatedCoreIds,omitempty"`
	StartTime        string  `json:"startTime,omitempty"`
	EndTime          string  `json:"endTime,omitempty"`
	ErrorMessage     string  `json:"errorMessage,omitempty"`
	AllParameters    string  `json:"allParameters,omitempty"`
}

// JobCreateRESTResponse holds the response to a job creation
type JobCreateRESTResponse struct {
	ID                 int64                       `json:"id"`
	Name               string                      `json:"name"`
	State              int                         `json:"state"`
	Priority           int                         `json:"priority"`
	Project            string                      `json:"project"`
	CreationTime       string                      `json:"creationTime"`
	SubmitTime         string                      `json:"submitTime,omitempty"`
	StartTime          string                      `json:"startTime,omitempty"`
	EndTime            string                      `json:"endTime,omitempty"`
	TotalAllocatedTime float64                     `json:"totalAllocatedTime,omitempty"`
	AllParameters      string                      `json:"allParameters,omitempty"`
	NodeType           ClusterNodeTypeRESTResponse `json:"nodeType"`
	Tasks              []TaskRESTResponse          `json:"tasks"`
}
