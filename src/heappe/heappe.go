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

// Client is the client interface to HEAppE service
type Client interface {
	CreateJob() (string, error)
}

// NewBasicAuthClient returns a client performing a basic user/pasword authentication
func NewBasicAuthClient(url, username, password string) Client {
	return &client{
		URL:      url,
		Username: username,
		Password: password,
	}
}

type client struct {
	URL      string
	Username string
	Password string
}

// CreateJob creates a HEAppE job
func (c *client) CreateJob() (string, error) {

	return "jobid", nil
}
