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

package heappecollector

import (
	"context"
	"encoding/json"
	"time"

	"github.com/laurentganne/yorc-heappe-plugin/v1/collectors"
	"github.com/laurentganne/yorc-heappe-plugin/v1/heappe"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
)

const (
	infrastructureType = "heappe"
)

type heappeUsageCollectorDelegate struct {
}

// NewInfraUsageCollectorDelegate creates a new slurm infra usage collector delegate for specific slurm infrastructure
func NewInfraUsageCollectorDelegate() collectors.InfraUsageCollectorDelegate {
	return &heappeUsageCollectorDelegate{}
}

// CollectInfo allows to collect usage info about defined infrastructure
func (h *heappeUsageCollectorDelegate) CollectInfo(ctx context.Context, cfg config.Configuration,
	taskID, infraName string) (map[string]interface{}, error) {

	locationMgr, err := locations.GetManager(cfg)
	if err != nil {
		return nil, err
	}

	locationProps, err := locationMgr.GetPropertiesForFirstLocationOfType(infrastructureType)
	if err != nil {
		return nil, err
	}

	userName := locationProps.GetString(heappe.LocationUserPropertyName)
	if userName == "" {
		return nil, errors.Errorf("No user defined in location")
	}

	heappeClient, err := heappe.GetClient(locationProps)
	if err != nil {
		return nil, err
	}

	// Getting corresponding user ID
	userID, err := getUserID(heappeClient, userName)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	endTimeStr := currentTime.Format(time.RFC3339)
	startTime := currentTime.AddDate(0, -1, 0)
	startTimeStr := startTime.Format(time.RFC3339)
	report, err := heappeClient.GetUserResourceUsageReport(userID, startTimeStr, endTimeStr)

	bytesVal, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(bytesVal, &result)

	if err != nil {
		log.Printf("Got error unmarshaling result: %+s\n", err.Error())

	}
	return result, err
}

func getUserID(client heappe.Client, userName string) (int64, error) {
	var userID int64

	adaptorUserGroups, err := client.ListAdaptorUserGroups()
	if err != nil {
		return userID, err
	}
	for _, adaptorUserGroup := range adaptorUserGroups {
		for _, adaptorUser := range adaptorUserGroup.Users {
			if adaptorUser.Username == userName {
				return adaptorUser.ID, err
			}
		}
	}

	return userID, errors.Errorf("Found no user with name %s", userName)
}
