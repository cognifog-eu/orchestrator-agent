/*
  OCM-DESCRIPTION-SERVICE
  Copyright © 2022-2024 EVIDEN

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

  This work has received funding from the European Union's HORIZON research
  and innovation programme under grant agreement No. 101070177.
*/

package models

import (
	"context"
	"etsn/server/ocm-description-service/utils/logs"
	"testing"

	"github.com/stretchr/testify/assert"

	workfake "open-cluster-management.io/api/client/work/clientset/versioned/fake"
	workv1 "open-cluster-management.io/api/work/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExecuteJob(t *testing.T) {
	t.Run("should generate a manifest work", func(t *testing.T) {
		j := MockCreateDeploymentJob()
		manifestWork := GenerateManifestWork(&j)
		assert.NotNil(t, manifestWork)
	})

	t.Run("should create a new deployment", func(t *testing.T) {
		j := MockCreateDeploymentJob()
		manifestWork := GenerateManifestWork(&j)
		jobClient := workfake.NewSimpleClientset()

		namespace := j.Namespace

		createdDeployment, err := MockCreateNewDeployment(jobClient, namespace, manifestWork)

		assert.NotNil(t, createdDeployment)
		assert.NoError(t, err)
	})

	t.Run("should handle deployment updates", func(t *testing.T) {
		updateTests := []struct {
			name    string
			subType RemediationType
		}{
			{name: "should scale up a deployment", subType: ScaleUp},
			{name: "should scale down a deployment", subType: ScaleDown},
			{name: "should scale in a deployment", subType: ScaleIn},
			{name: "should scale out a deployment", subType: ScaleOut},
		}

		for _, tt := range updateTests {
			t.Run(tt.name, func(t *testing.T) {
				j := MockUpdateJob(tt.subType)
				manifestWork := GenerateManifestWork(&j)
				jobClient := workfake.NewSimpleClientset()
				namespace := j.Namespace

				createdManifestWork, err := MockCreateNewDeployment(jobClient, namespace, manifestWork)
				assert.NoError(t, err)
				assert.NotNil(t, createdManifestWork)

				fetchedManifestWork, err := MockGetManifestWork(jobClient, namespace, createdManifestWork.Name)
				assert.NoError(t, err)
				assert.NotNil(t, fetchedManifestWork)

				// TODO: Implement UpdateDeploymentAttributes
				//updatedManifests, err := UpdateDeploymentAttributes(fetchedManifestWork.Spec.Workload.Manifests, tt.subType)
				// assert.NoError(t, err)
				// assert.NotNil(t, updatedManifests)

				// fetchedManifestWork.Spec.Workload.Manifests = updatedManifests

				assert.NoError(t, MockUpdateManifestWork(jobClient, fetchedManifestWork))
			})
		}
	})
}

func MockGetManifestWork(jobClient *workfake.Clientset, namespace, name string) (*workv1.ManifestWork, error) {
	manifestWork, err := jobClient.WorkV1().ManifestWorks(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		logs.Logger.Println("Error fetching manifest work:", err)
	}
	return manifestWork, err
}

func MockCreateNewDeployment(jobClient *workfake.Clientset, namespace string, manifestWork *workv1.ManifestWork) (*workv1.ManifestWork, error) {
	return jobClient.WorkV1().ManifestWorks(namespace).Create(context.TODO(), manifestWork, metav1.CreateOptions{})
}

func MockUpdateManifestWork(jobClient *workfake.Clientset, manifestWork *workv1.ManifestWork) error {
	_, err := jobClient.WorkV1().ManifestWorks(manifestWork.Namespace).Update(context.TODO(), manifestWork, metav1.UpdateOptions{})
	return err
}

func MockUpdateJob(subType RemediationType) Job {
	panic("not implemented")
}

func MockCreateDeploymentJob() Job {
	panic("not implemented")

}
