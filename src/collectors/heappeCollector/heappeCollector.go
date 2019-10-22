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

package heappeCollector

import (
	"context"

	"github.com/laurentganne/yorc-heappe-plugin/v1/collectors"
	"github.com/laurentganne/yorc-heappe-plugin/v1/heappe"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/locations"
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

	_, err = heappe.GetClient(locationProps)
	if err != nil {
		return nil, err
	}
	return nil, err
}
