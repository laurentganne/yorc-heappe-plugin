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

package main

import (
	"context"

	"github.com/laurentganne/yorc-heappe-plugin/v1/collectors"
	"github.com/laurentganne/yorc-heappe-plugin/v1/collectors/heappeCollector"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/log"
	"github.com/ystia/yorc/v4/prov"
)

// infraUsageCollector represents a structure responsible for collecting multiple infra usage information
type infraUsageCollector struct {
	collectorDelegates map[string]collectors.InfraUsageCollectorDelegate
}

// NewInfraUsageCollector creates a new infra usage collector and delegates to suitable delegate for each infrastructure
func newInfraUsageCollector() prov.InfraUsageCollector {
	// List all the collectors delegates implemented in the plugin
	collectorDelegates := make(map[string]collectors.InfraUsageCollectorDelegate)
	collectorDelegates[heappeInfrastructureType] = heappeCollector.NewInfraUsageCollectorDelegate()

	return &infraUsageCollector{
		collectorDelegates: collectorDelegates,
	}
}

// GetUsageInfo returns infrastructure usage information
func (i *infraUsageCollector) GetUsageInfo(ctx context.Context, cfg config.Configuration,
	taskID, infraName string) (map[string]interface{}, error) {

	log.Printf("Retrieving infrastructure usage info for infra:%q", infraName)
	collDelegate, exist := i.collectorDelegates[infraName]
	if !exist {
		return nil, errors.Errorf("No infra collector delegate found for the infrastructure:%s", infraName)
	}
	return collDelegate.CollectInfo(ctx, cfg, taskID, infraName)
}
