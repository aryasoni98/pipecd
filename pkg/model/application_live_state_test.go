// Copyright 2024 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplicationLiveStateSnapshot_DetermineAppHealthStatus(t *testing.T) {
	testcases := []struct {
		name     string
		snapshot *ApplicationLiveStateSnapshot
		want     ApplicationLiveStateSnapshot_Status
	}{
		{
			name: "kubernetes: healthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_KUBERNETES,
				Kubernetes: &KubernetesApplicationLiveState{
					Resources: []*KubernetesResourceState{{HealthStatus: KubernetesResourceState_HEALTHY}},
				},
			},
			want: ApplicationLiveStateSnapshot_HEALTHY,
		},
		{
			name: "kubernetes: unhealthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_KUBERNETES,
				Kubernetes: &KubernetesApplicationLiveState{
					Resources: []*KubernetesResourceState{
						{HealthStatus: KubernetesResourceState_HEALTHY},
						{HealthStatus: KubernetesResourceState_OTHER},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_OTHER,
		},
		{
			name: "kubernetes: unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_KUBERNETES,
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
		{
			name: "cloudrun: healthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_CLOUDRUN,
				Cloudrun: &CloudRunApplicationLiveState{
					Resources: []*CloudRunResourceState{{HealthStatus: CloudRunResourceState_HEALTHY}},
				},
			},
			want: ApplicationLiveStateSnapshot_HEALTHY,
		},
		{
			name: "cloudrun: unhealthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_CLOUDRUN,
				Cloudrun: &CloudRunApplicationLiveState{
					Resources: []*CloudRunResourceState{
						{HealthStatus: CloudRunResourceState_HEALTHY},
						{HealthStatus: CloudRunResourceState_OTHER},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_OTHER,
		},
		{
			name: "cloudrun: unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_CLOUDRUN,
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
		{
			name: "terraform: unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_TERRAFORM,
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
		{
			name: "lambda: unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_LAMBDA,
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
		{
			name: "lambda: healthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_LAMBDA,
				Lambda: &LambdaApplicationLiveState{
					Resources: []*LambdaResourceState{{HealthStatus: LambdaResourceState_HEALTHY}},
				},
			},
			want: ApplicationLiveStateSnapshot_HEALTHY,
		},
		{
			name: "lambda: unhealthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_LAMBDA,
				Lambda: &LambdaApplicationLiveState{
					Resources: []*LambdaResourceState{
						{HealthStatus: LambdaResourceState_HEALTHY},
						{HealthStatus: LambdaResourceState_OTHER},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_OTHER,
		},
		{
			name: "lambda: unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_LAMBDA,
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
		{
			name: "ecs: healthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_ECS,
				Ecs: &ECSApplicationLiveState{
					Resources: []*ECSResourceState{{HealthStatus: ECSResourceState_HEALTHY}},
				},
			},
			want: ApplicationLiveStateSnapshot_HEALTHY,
		},
		{
			name: "ecs: unhealthy",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_ECS,
				Ecs: &ECSApplicationLiveState{
					Resources: []*ECSResourceState{
						{HealthStatus: ECSResourceState_HEALTHY},
						{HealthStatus: ECSResourceState_OTHER},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_OTHER,
		},
		{
			name: "ecs: unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				Kind: ApplicationKind_ECS,
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.snapshot.DetermineAppHealthStatus()
			assert.Equal(t, tc.want, tc.snapshot.HealthStatus)
		})
	}
}

func TestApplicationLiveStateSnapshot_DetermineApplicationHealthStatus(t *testing.T) {
	testcases := []struct {
		name     string
		snapshot *ApplicationLiveStateSnapshot
		want     ApplicationLiveStateSnapshot_Status
	}{
		{
			name: "healthy: all resources are healthy",
			snapshot: &ApplicationLiveStateSnapshot{
				ApplicationLiveState: &ApplicationLiveState{
					Resources: []*ResourceState{
						{HealthStatus: ResourceState_HEALTHY},
						{HealthStatus: ResourceState_HEALTHY},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_HEALTHY,
		},
		{
			name: "healthy: no resources",
			snapshot: &ApplicationLiveStateSnapshot{
				ApplicationLiveState: &ApplicationLiveState{},
			},
			want: ApplicationLiveStateSnapshot_HEALTHY,
		},
		{
			name: "unhealthy: one of the resources is unhealthy",
			snapshot: &ApplicationLiveStateSnapshot{
				ApplicationLiveState: &ApplicationLiveState{
					Resources: []*ResourceState{
						{HealthStatus: ResourceState_HEALTHY},
						{HealthStatus: ResourceState_UNHEALTHY},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_UNHEALTHY,
		},
		{
			name: "unhealthy: prioritize unhealthy over unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				ApplicationLiveState: &ApplicationLiveState{
					Resources: []*ResourceState{
						{HealthStatus: ResourceState_UNKNOWN},
						{HealthStatus: ResourceState_UNHEALTHY},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_UNHEALTHY,
		},
		{
			name: "unknown: one of the resources is unknown",
			snapshot: &ApplicationLiveStateSnapshot{
				ApplicationLiveState: &ApplicationLiveState{
					Resources: []*ResourceState{
						{HealthStatus: ResourceState_HEALTHY},
						{HealthStatus: ResourceState_UNKNOWN},
					},
				},
			},
			want: ApplicationLiveStateSnapshot_UNKNOWN,
		},
		{
			name:     "unknown: nil application live state",
			snapshot: &ApplicationLiveStateSnapshot{},
			want:     ApplicationLiveStateSnapshot_UNKNOWN,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tc.snapshot.DetermineApplicationHealthStatus()
			assert.Equal(t, tc.want, tc.snapshot.HealthStatus)
		})
	}
}
