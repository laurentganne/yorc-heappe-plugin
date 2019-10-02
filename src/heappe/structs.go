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

package heappe

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
	TemplateParameterValues []CommandTemplateParameterValue `json:"templateParameterValues, omitempty"`
	MinCores                int                             `json:"minCores"`
	MaxCores                int                             `json:"maxCores"`
	WalltimeLimit           int                             `json:"walltimeLimit"`
	StandardOutputFile      string                          `json:"standardOutputFile"`
	StandardErrorFile       string                          `json:"standardErrorFile"`
	ProgressFile            string                          `json:"progressFile"`
	LogFile                 string                          `json:"logFile"`
	EnvironmentVariables    []EnvironmentVariable           `json:"environmentVariables, omitempty"`
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
