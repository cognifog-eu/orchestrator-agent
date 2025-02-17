/*
Copyright 2023-2024 Bull SAS

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package models

import (
	"bytes"
	"context"
	"encoding/json"
	"etsn/server/ocm-description-service/utils/logs"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	//yaml "gopkg.in/yaml.v2"
	"github.com/google/uuid"
	"github.com/mattn/go-shellwords"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clustermanager "open-cluster-management.io/api/client/operator/clientset/versioned/typed/operator/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	yamlEncode "sigs.k8s.io/yaml"

	workclient "open-cluster-management.io/api/client/work/clientset/versioned"
)

var (
	jobmanagerBaseURL    = os.Getenv("JOBMANAGER_URL")
	clientset            *kubernetes.Clientset
	clientsetWorkOper    workclient.Interface
	clientsetClusterOper clusterclient.Interface
	clientOperator       *clustermanager.OperatorV1Client
	JobTypeToString      = map[JobType]string{
		CreateDeployment:  "CreateDeployment",
		UpdateDeployment:  "UpdateDeployment",
		DeleteDeployment:  "DeleteDeployment",
		ReplaceDeployment: "ReplaceDeployment",
	}
)

// Base entities with UUID and UINT
type BaseUUID struct {
	//Metadata
	ID string `json:"id"`
}

type BaseUINT struct {
	//Metadata
	ID uint32 `json:"id"`
}

type Job struct {
	BaseUUID
	JobGroupID   string           `json:"job_group_id,omitempty" validate:"omitempty,uuid4"`
	OwnerID      string           `json:"owner_id,omitempty" validate:"omitempty,uuid4"`
	Type         JobType          `json:"type"`
	SubType      RemediationType  `json:"sub_type,omitempty"`
	State        JobState         `json:"state"`
	Target       Target           `json:"targets,omitempty" validate:"omitempty"`
	Orchestrator OrchestratorType `json:"orchestrator"`
	Instruction  *Instruction     `json:"instruction,omitempty"`
	Resource     *Resource        `json:"resource,omitempty"`
	Namespace    string           `json:"namespace,omitempty" validate:"omitempty"`
}

type InstructionBase struct {
	ComponentName string `json:"componentName,omitempty" yaml:"name,omitempty"`
	Type          string `json:"type,omitempty" yaml:"type,omitempty"`
}

type Instruction struct {
	BaseUUID
	InstructionBase
	JobID    string    `json:"job_id,omitempty"`
	Contents []Content `json:"contents,omitempty"`
}

type Content struct {
	BaseUINT
	Name          string `json:"name"`
	InstructionID string `json:"instruction_id,omitempty"`
	Yaml          string `json:"yaml"`
}

// Incompliance entity
type Remediation struct {
	BaseUUID
	RemediationType   RemediationType    `gorm:"type:text" json:"remediationType" validate:"required"`
	Status            RemediationStatus  `gorm:"type:text" default:"Pending" json:"remediationStatus" validate:"required"`
	RemediationTarget *RemediationTarget `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"remediationTarget,omitempty" validate:"omitempty"`
	ResourceID        string             `gorm:"type:char(36);not null" json:"resource_id" validate:"omitempty,uuid4"`
}

type RemediationTarget struct {
	BaseUUID
	RemediationID string `gorm:"type:char(36);not null" json:"remediation_id" validate:"omitempty,uuid4"`
	Container     string `gorm:"type:text" json:"container,omitempty" validate:"omitempty"`
	PodUID        string `gorm:"type:text" json:"pod_uid,omitempty" validate:"omitempty"`
	Pod           string `gorm:"type:text" json:"pod,omitempty" validate:"omitempty"`
	Node          string `gorm:"type:text" json:"node,omitempty" validate:"omitempty"`
	Namespace     string `gorm:"type:text" json:"namespace,omitempty" validate:"omitempty"`
	Command       string `gorm:"type:text" json:"command,omitempty" validate:"omitempty"`
}

type StringMap map[string]string

type JobState string
type RemediationStatus string

// type ResourceState string
type ConditionStatus string

type Resource struct {
	ResourceUID  string             `json:"resource_uuid,omitempty"`
	JobID        string             `json:"job_id"`
	ResourceName string             `json:"resource_name,omitempty"`
	Conditions   []metav1.Condition `json:"conditions,omitempty"`
	Remediations []Remediation      `json:"remediations,omitempty" validate:"omitempty"`
}

type PlainManifest struct {
	BaseUINT   `json:"-"`
	JobID      string `json:"-"`
	YamlString string `json:"yamlString"`
}

type Target struct {
	BaseUINT
	JobID        string           `json:"-"`
	ClusterName  string           `json:"cluster_name"`
	NodeName     string           `json:"node_name,omitempty"`
	Orchestrator OrchestratorType `json:"orchestrator"`
}

// Type declarations
type (
	State            int
	JobType          string
	OrchestratorType string
	RemediationType  string
)

// Constants for OrchestratorType and RemediationType
const (
	OCM        OrchestratorType = "ocm"
	NUVLA      OrchestratorType = "nuvla"
	ScaleUp    RemediationType  = "scale-up"
	ScaleDown  RemediationType  = "scale-down"
	ScaleOut   RemediationType  = "scale-out"
	ScaleIn    RemediationType  = "scale-in"
	Patch      RemediationType  = "patch"
	Reallocate RemediationType  = "reallocate"
	Replace    RemediationType  = "replace"
	Secure     RemediationType  = "secure"
)

// JobState Enum
const (
	Created     JobState = "Created"
	Progressing JobState = "Progressing"
	Finished    JobState = "Finished"
	Degraded    JobState = "Degraded"
)

// JobType Enum
const (
	CreateDeployment  JobType = "CreateDeployment"
	DeleteDeployment  JobType = "DeleteDeployment"
	UpdateDeployment  JobType = "UpdateDeployment"
	ReplaceDeployment JobType = "ReplaceDeployment"
)

// JobStatus represents the status of a Kubernetes Job.
type KubernetesJobStatus struct {
	JobSucceeded int
	JobFailed    int
}

// Configuration and Initialization
// ------------------------------------------------)

var kubeconfig *string

// InClusterConfig sets up Kubernetes client configurations for in-cluster and out-of-cluster environments.
func InClusterConfig() error {
	config, err := rest.InClusterConfig()

	// Outside of the cluster for development
	if err != nil {
		//panic(err.Error())
		logs.Logger.Println("The home folder is: ", homedir.HomeDir())
		if home := homedir.HomeDir(); home != "" {
			if kubeconfig == nil {
				if flag.Lookup("kubeconfig") == nil {
					kubeconfig = flag.String("kubeconfig", "/home/dev/.kube/config-k3s-master1", "absolute path to the kubeconfig file")
				} else {
					kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
				}
			}
		} else {
			if kubeconfig == nil {
				if flag.Lookup("kubeconfig") == nil {
					kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
				}
			}
		}
		flag.Parse()
		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}

	// creates the clientset
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientsetWorkOper, err = workclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientsetClusterOper, err = clusterclient.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	clientOperator, err = clustermanager.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	return err
}

// Job Execution and Management
// ------------------------------------------------

// Execute executes the job based on its type, such as creating, updating, or deleting a deployment.
func Execute(j *Job) (*Job, error) {
	jobType := getJobTypeString(j.Type)
	logs.Logger.Println("Executing job type:", jobType)

	switch j.Type {
	case CreateDeployment:
		return createDeployment(j)
	case UpdateDeployment:
		return updateDeployment(j)
	case DeleteDeployment:
		return deleteDeployment(j)
	default:
		err := fmt.Errorf("job type not supported: %s", jobType)
		logs.Logger.Println(err)
		return nil, err
	}
}

// createDeployment creates a new deployment for the given job and updates the job's resource details.
func createDeployment(j *Job) (*Job, error) {
	return createAndApplyManifestWork(j)
}

func createAndApplyManifestWork(j *Job) (*Job, error) {
	logs.Logger.Println("Creating Work for Job:", j.ID)

	// Initialize slice for updated manifests
	updatedManifests := []workv1.Manifest{}

	// Create the ManifestWork
	mw, err := createManifestWork(j)
	if err != nil {
		logErrorAndSetJobState("Error creating ManifestWork", j, Degraded)
		return nil, err
	}

	// Extract namespace and resource UUID
	namespace := mw.Namespace
	resUUID := string(mw.GetUID())

	if resUUID != "" {
		logs.Logger.Println("ManifestWork UID: ", resUUID)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		appliedManifestWork, err := waitForAppliedManifestWork(namespace, mw.Name, ctx)
		if err != nil {
			logs.Logger.Println("Error obtaining applied ManifestWork status:", err)
			j.State = Degraded
			return nil, err
		}
		j.UpdateJobResource(appliedManifestWork)

		for _, manifest := range appliedManifestWork.Spec.Workload.Manifests {
			obj, err := decodeAndUnmarshalManifest(manifest)
			if err != nil {
				logs.Logger.Println("Error decoding manifest:", err)
				continue
			}

			if obj.GetObjectKind().GroupVersionKind().Kind == "Namespace" {
				updatedManifests = append(updatedManifests, manifest)
				logs.Logger.Println("Skipping Namespace manifest")
				continue
			}

			// Access object metadata
			metaObj, err := meta.Accessor(obj)
			if err != nil {
				logs.Logger.Fatalf("Failed to get metadata accessor: %v", err)
			}
			annotations := metaObj.GetAnnotations()

			annotations["jobmanager.cognifog.eu/manifest"] = resUUID // manifestwork UID
			metaObj.SetAnnotations(annotations)

			rawExtension := runtime.RawExtension{Object: obj}
			manifest := workv1.Manifest{RawExtension: rawExtension}

			updatedManifests = append(updatedManifests, manifest)
		}

		// Apply the patch to update the manifest work with resource annotation
		return applyPatch(j, updatedManifests)
	}

	// Apply the patch to update the manifest work with annotations and feedback rules
	//return applyPatch(j, updatedManifests)ç
	return j, err
}

// bluegreen deployment strategy
func replaceDeployment(j *Job) (*Job, error) {
	logs.Logger.Println("Replacing Work for Job:", j.ID)

	// Fetch old deployment
	oldManifestWork, err := fetchManifestWork(j.Target.ClusterName, j.Resource.ResourceName, nil)
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return nil, err
	}

	updatedManifests := []workv1.Manifest{}

	for _, content := range j.Instruction.Contents {
		obj, err := decodeYAMLToObject(content.Yaml)
		if err != nil {
			logs.Logger.Println("Error unmarshaling manifest:", err)
			continue
		}
		updateNamespaceAndAnnotations(obj, j.Namespace, j.Resource.ResourceName, j.JobGroupID)
		rawExtension := runtime.RawExtension{Object: obj}
		manifest := workv1.Manifest{RawExtension: rawExtension}
		// usar para el replace deployment
		updatedManifests = append(updatedManifests, manifest)
	}

	oldManifestWork.Spec.Workload.Manifests = updatedManifests

	updatedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Update(context.TODO(), oldManifestWork, metav1.UpdateOptions{})

	if err != nil {
		logErrorAndSetJobState("Error updating ManifestWork", j, Degraded)
		return nil, err
	}

	j.UpdateJobResource(updatedManifestWork)

	return j, nil
}

// Used in remediation actions
// updateDeployment updates an existing deployment for the given job and updates the job's resource details.
func updateDeployment(j *Job) (*Job, error) {
	logs.Logger.Println("Updating work for Job:", j.ID)
	switch j.SubType {

	case ScaleUp, ScaleDown, ScaleOut, ScaleIn:
		// Stateful remediation action
		return updateDeploymentAttributes(j)
	case Reallocate:
		// Stateless remediation action
		return deleteDeployment(j)
	case Patch:
		// Stateful remediation action
		return patchDeployment(j)
	case Replace:
		// Stateless remediation action
		return replaceDeployment(j)
	case Secure:
		// Stateful remediation action
		return applySecurityAction(j)
	// case Secure:
	// 	return secureDeployment(j)
	default:
		logErrorAndSetJobState("Job Sub Type does not exist", j, Degraded)
		return nil, fmt.Errorf("job sub type does not exist: %v", j.SubType)
	}
}

// deleteDeployment deletes the deployment associated with the given job and clears the job's resource details.
func deleteDeployment(j *Job) (*Job, error) {
	zero := int64(0)
	logs.Logger.Println("Deleting deployment for Job:", j.ID)

	err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Delete(context.TODO(), j.Resource.ResourceName, metav1.DeleteOptions{GracePeriodSeconds: &zero})
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return j, err
	}

	logs.Logger.Printf("Successfully deleted deployment for Job: %s\n", j.ID)
	// j.State = Applied --> this is a resource state not a job state!
	j.Resource.Conditions = append(j.Resource.Conditions, metav1.Condition{
		Type:               "Deleted",
		Status:             "True",
		Reason:             "Deleted",
		Message:            "Deployment deleted successfully",
		ObservedGeneration: 0,
		LastTransitionTime: metav1.Time{
			Time: time.Now(),
		},
	})
	return j, nil
}

// OCM Manifest Work Operations
// ------------------------------------------------
// createManifestWork creates a manifest work for the given job in the specified cluster.
func createManifestWork(j *Job) (*workv1.ManifestWork, error) {
	// We need to create a new resource for the job
	j.Resource = &Resource{
		JobID:        j.ID,
		ResourceName: j.Instruction.ComponentName,
		Conditions: []metav1.Condition{
			{
				Type:               "Progressing",
				Status:             "True",
				ObservedGeneration: 1,
				LastTransitionTime: metav1.Time{Time: time.Now()},
				Reason:             "Job Promoted",
				Message:            "Job promoted for execution",
			},
		},
	}
	manifestWork := GenerateManifestWork(j)
	createdManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Create(context.TODO(), manifestWork, metav1.CreateOptions{})
	if err != nil {
		logErrorAndSetJobState("Error creating ManifestWork", j, Degraded)
		return nil, fmt.Errorf("error creating ManifestWork: %v", err)
	}
	return createdManifestWork, nil
}

// fetchManifestWork retrieves the manifest work object from the specified namespace and name.
func fetchManifestWork(namespace, manifestWorkName string, ctx context.Context) (*workv1.ManifestWork, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	manifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Get(ctx, manifestWorkName, metav1.GetOptions{})
	if err != nil || manifestWork == nil {
		return nil, fmt.Errorf("error obtaining applied ManifestWork status: %v", err)
	}
	return manifestWork, nil
}

// waitForAppliedManifestWork polls for the applied ManifestWork until it is found or the context times out.
func waitForAppliedManifestWork(namespace, name string, ctx context.Context) (*workv1.ManifestWork, error) {
	var appliedManifestWork *workv1.ManifestWork
	var err error

	for {
		appliedManifestWork, err = fetchManifestWork(namespace, name, ctx)
		if err != nil {
			logs.Logger.Println("Error obtaining applied ManifestWork status:", err)
		}

		if appliedManifestWork != nil && len(appliedManifestWork.Status.Conditions) > 0 {
			break
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context timed out while waiting for applied ManifestWork status")
		case <-time.After(500 * time.Millisecond): // Poll interval
		}
	}

	return appliedManifestWork, nil
}

// GenerateManifestWork generates a manifest work object for the given job.
func GenerateManifestWork(j *Job) *workv1.ManifestWork {
	work := workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Resource.ResourceName + "-",
			Namespace:    j.Target.ClusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{},
		},
	}

	namespaceManifest, err := generateNamespaceManifest(j.Namespace)
	if err != nil {
		logs.Logger.Println("Error generating namespace manifest:", err)
	}
	work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, namespaceManifest)

	for _, content := range j.Instruction.Contents {
		obj, err := decodeYAMLToObject(content.Yaml)
		if err != nil {
			logs.Logger.Println("Error unmarshaling manifest:", err)
			continue
		}
		updateNamespaceAndAnnotations(obj, j.Namespace, j.Resource.ResourceName, j.JobGroupID)
		rawExtension := runtime.RawExtension{Object: obj}
		manifest := workv1.Manifest{RawExtension: rawExtension}
		logs.Logger.Print("------Inside GenerateManifestWork----------")
		logs.Logger.Print("Manifest Kind: ", manifest.RawExtension.Object.GetObjectKind().GroupVersionKind().Kind)
		logs.Logger.Print("------Inside GenerateManifestWork----------")
		// usar para el replace deployment
		work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, manifest)
	}

	return &work
}

// updateDeploymentAttributes updates the attributes of the manifests for a deployment based on the remediation type.
func updateDeploymentAttributes(j *Job) (*Job, error) {
	subType := j.SubType

	manifestWork, err := fetchManifestWork(j.Target.ClusterName, j.Resource.ResourceName, nil)
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return nil, fmt.Errorf("error obtaining applied ManifestWork status: %v", err)
	}

	updatedManifests, err := processManifests(manifestWork.Spec.Workload.Manifests, func(obj runtime.Object) (*workv1.Manifest, error) {
		switch subType {
		case ScaleUp, ScaleDown:
			return updateReplicaCount(obj, subType)
		case ScaleOut, ScaleIn:
			return updateResourceRequirements(obj, subType)
		default:
			return nil, fmt.Errorf("unsupported subType: %v", subType)
		}
	})
	if err != nil {
		return nil, err
	}

	return applyPatch(j, updatedManifests)
}

// patchDeployment updates deployment manifests with namespace and annotations.
func patchDeployment(j *Job) (*Job, error) {
	_, err := fetchManifestWork(j.Target.ClusterName, j.Resource.ResourceName, nil)
	if err != nil {
		logErrorAndSetJobState("Error obtaining applied ManifestWork status", j, Degraded)
		return nil, fmt.Errorf("error obtaining applied ManifestWork status: %v", err)
	}

	namespaceManifest, err := generateNamespaceManifest(j.Namespace)
	if err != nil {
		logErrorAndSetJobState("Error generating namespace manifest", j, Degraded)
		return nil, fmt.Errorf("error generating namespace manifest: %v", err)
	}

	updatedManifests := []workv1.Manifest{namespaceManifest}
	for _, content := range j.Instruction.Contents {
		obj, err := decodeYAMLToObject(content.Yaml)
		if err != nil {
			logs.Logger.Println("Error unmarshaling manifest:", err)
			continue
		}
		updateNamespaceAndAnnotations(obj, j.Namespace, j.Resource.ResourceName, j.JobGroupID)
		updatedManifests = append(updatedManifests, workv1.Manifest{RawExtension: runtime.RawExtension{Object: obj}})
	}

	return applyPatch(j, updatedManifests)
}

// applyPatch applies the patch with the updated manifests to the ManifestWork resource.
func applyPatch(j *Job, updatedManifests []workv1.Manifest) (*Job, error) {
	// Create a patch that updates the Workload.Manifests field
	patchData := map[string]interface{}{
		"spec": map[string]interface{}{
			"workload": map[string]interface{}{
				"manifests": updatedManifests,
			},
		},
	}

	// Convert the patchData to JSON
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		logErrorAndSetJobState("Error creating JSON patch data", j, Degraded)
		return nil, fmt.Errorf("error creating JSON patch data: %v", err)
	}

	// Use the Patch method to apply the update
	updatedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Patch(
		context.TODO(),
		j.Resource.ResourceName,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)

	if err != nil {
		logErrorAndSetJobState("Error patching ManifestWork", j, Degraded)
		return nil, fmt.Errorf("error patching ManifestWork: %v", err)
	}

	// Update the job resource with the patched ManifestWork
	j.UpdateJobResource(updatedManifestWork)

	return j, nil
}

// processManifests processes each manifest by encoding/decoding and applying the provided updateFunc.
func processManifests(manifests []workv1.Manifest, updateFunc func(runtime.Object) (*workv1.Manifest, error)) ([]workv1.Manifest, error) {
	updatedManifests := make([]workv1.Manifest, 0, len(manifests))
	for _, manifest := range manifests {
		obj, err := decodeAndUnmarshalManifest(manifest)
		if err != nil {
			return nil, err
		}
		updatedManifest, err := updateFunc(obj)
		if err != nil {
			return nil, fmt.Errorf("error updating manifest: %v", err)
		}
		if updatedManifest != nil {
			updatedManifests = append(updatedManifests, *updatedManifest)
		} else {
			updatedManifests = append(updatedManifests, manifest)
		}
	}
	return updatedManifests, nil
}

// decodeAndUnmarshalManifest decodes and unmarshals a manifest.
func decodeAndUnmarshalManifest(manifest workv1.Manifest) (runtime.Object, error) {
	yamlBytes, err := yamlEncode.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("error encoding manifest: %v", err)
	}
	obj, err := decodeYAMLToObject(string(yamlBytes))
	if err != nil {
		return nil, fmt.Errorf("error decoding manifest: %v", err)
	}
	return obj, nil
}

// applySecurityAction applies a security action on the specified pod/container in the manifests.
func applySecurityAction(j *Job) (*Job, error) {

	remediationLength := len(j.Resource.Remediations)
	if remediationLength == 0 {
		logErrorAndSetJobState("No remediations found", j, Degraded)
		return nil, fmt.Errorf("no remediations found")
	}

	remediationTarget := j.Resource.Remediations[remediationLength-1].RemediationTarget
	targetPod := remediationTarget.Pod
	targetContainer := remediationTarget.Container
	command := remediationTarget.Command
	namespace := j.Namespace
	jobName := fmt.Sprintf("%s-job-%d", j.Resource.ResourceName, time.Now().Unix())
	saName := fmt.Sprintf("%s-job-sa", j.Resource.ResourceName)
	roleName := fmt.Sprintf("%s-job-role", j.Resource.ResourceName)
	roleBindingName := fmt.Sprintf("%s-job-rolebinding", j.Resource.ResourceName)

	// Generate manifests
	serviceAccount, err := generateServiceAccountManifest(namespace, saName)
	if err != nil {
		logErrorAndSetJobState("Error generating ServiceAccount manifest", j, Degraded)
		j.Resource.Remediations[remediationLength-1].Status = "Failed"
		return nil, fmt.Errorf("error generating ServiceAccount manifest: %v", err)
	}

	role, err := generateRoleManifest(namespace, roleName)
	if err != nil {
		logErrorAndSetJobState("Error generating Role manifest", j, Degraded)
		j.Resource.Remediations[remediationLength-1].Status = "Failed"
		return nil, fmt.Errorf("error generating Role manifest: %v", err)
	}

	roleBinding, err := generateRoleBindingManifest(namespace, roleBindingName, roleName, saName)
	if err != nil {
		logErrorAndSetJobState("Error generating RoleBinding manifest", j, Degraded)
		j.Resource.Remediations[remediationLength-1].Status = "Failed"
		return nil, fmt.Errorf("error generating RoleBinding manifest: %v", err)
	}

	securityJob, err := generateKubernetesJobManifest(targetPod, targetContainer, command, namespace, saName, jobName)
	if err != nil {
		logErrorAndSetJobState("Error generating Kubernetes Job manifest", j, Degraded)
		j.Resource.Remediations[remediationLength-1].Status = "Failed"
		return nil, fmt.Errorf("error generating Kubernetes Job manifest: %v", err)
	}

	work := workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.Resource.ResourceName + "-job-",
			Namespace:    j.Target.ClusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{},
		},
	}

	work.Spec.Workload.Manifests = append(work.Spec.Workload.Manifests, serviceAccount, role, roleBinding, securityJob)
	work.Spec.ManifestConfigs = append(work.Spec.ManifestConfigs, workv1.ManifestConfigOption{
		ResourceIdentifier: workv1.ResourceIdentifier{
			Group:     "batch",
			Resource:  "jobs",
			Namespace: namespace,
			Name:      jobName,
		},
		FeedbackRules: []workv1.FeedbackRule{
			{
				Type: workv1.WellKnownStatusType,
			},
		},
		UpdateStrategy: &workv1.UpdateStrategy{
			Type: workv1.UpdateStrategyTypeServerSideApply,
		}})

	createdManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Target.ClusterName).Create(context.TODO(), &work, metav1.CreateOptions{})

	if err != nil {
		logErrorAndSetJobState("Error creating ManifestWork", j, Degraded)
		j.Resource.Remediations[remediationLength-1].Status = "Failed"
		return nil, fmt.Errorf("error creating ManifestWork: %v", err)
	}

	go MonitorJobAndCleanup(j.Target.ClusterName, createdManifestWork.Name, j.Namespace)

	j.Resource.Remediations[remediationLength-1].Status = "Applied"
	return j, nil
}

// generateKubernetesJob generates a Kubernetes Job object for the given job.
func generateKubernetesJobManifest(targetPod, targetContainer, command, namespace, serviceAccountName, jobName string) (workv1.Manifest, error) {
	yamlTemplate := `apiVersion: batch/v1
kind: Job
metadata:
  name: {{ .JobName }}
  namespace: {{ .Namespace }}
spec:
  backoffLimit: {{ .BackoffLimit }}
  template:
    spec:
      serviceAccountName: {{ .ServiceAccountName }}
      containers:
      - name: {{ .ContainerName }}
        image: bitnami/kubectl:latest
        command:
        - "kubectl"
        - "exec"
        - "{{ .TargetPod }}"
        - "-c"
        - "{{ .TargetContainer }}"
        - "--"
        {{- range .Command }}
        - "{{ . }}"
        {{- end }}
      restartPolicy: Never`

	tmpl, err := template.New("job").Parse(yamlTemplate)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing template: %v", err)
	}

	// Parse the command into arguments
	parser := shellwords.NewParser()
	parsedCommand, err := parser.Parse(command)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing command: %v", err)
	}

	jobPayload := struct {
		JobName            string
		ServiceAccountName string
		Namespace          string
		ContainerName      string
		TargetPod          string
		TargetContainer    string
		Command            []string
		BackoffLimit       int32
	}{
		JobName:            jobName,
		ServiceAccountName: serviceAccountName,
		Namespace:          namespace,
		ContainerName:      "kubectl",
		TargetPod:          targetPod,
		TargetContainer:    targetContainer,
		Command:            parsedCommand,
		BackoffLimit:       4,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, jobPayload); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error executing template: %v", err)
	}

	var manifest workv1.Manifest
	if err := yaml.Unmarshal(buf.Bytes(), &manifest); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error unmarshaling manifest: %v", err)
	}

	return manifest, nil
}

// MonitorJobAndCleanup periodically checks the Job's status and deletes the ManifestWork upon completion.
func MonitorJobAndCleanup(clusterName, manifestWorkName, namespace string) {

	ticker := time.NewTicker(30 * time.Second)
	zero := int64(0)
	timeout := time.After(10 * time.Minute)

	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			manifestWork, err := waitForAppliedManifestWork(clusterName, manifestWorkName, nil)
			if err != nil {
				logs.Logger.Printf("Error retrieving ManifestWork: %v\n", err)
				continue
			}

			jobStatus, err := GetJobStatusFromManifestWork(manifestWork)
			if err != nil {
				logs.Logger.Printf("Error retrieving Job status: %v\n", err)
				continue
			}
			switch {
			case jobStatus.JobSucceeded > 0:
				logs.Logger.Printf("Job %s succeeded.\n", manifestWorkName)
				err := clientsetWorkOper.WorkV1().ManifestWorks(clusterName).Delete(context.TODO(), manifestWorkName, metav1.DeleteOptions{GracePeriodSeconds: &zero})
				if err != nil {
					logs.Logger.Println("Error deleting ManifestWork:", err)
					return
				}
				return
			case jobStatus.JobFailed > 0:
				logs.Logger.Printf("Job %s failed.\n", manifestWorkName)
				err := clientsetWorkOper.WorkV1().ManifestWorks(clusterName).Delete(context.TODO(), manifestWorkName, metav1.DeleteOptions{GracePeriodSeconds: &zero})
				if err != nil {
					logs.Logger.Println("Error deleting ManifestWork:", err)
					return
				}
				return
			default:
				logs.Logger.Printf("Job %s is still running...\n", manifestWorkName)
			}

		case <-timeout:
			logs.Logger.Printf("Timeout reached while waiting for Job %s to complete.\n", manifestWorkName)
			err := clientsetWorkOper.WorkV1().ManifestWorks(clusterName).Delete(context.TODO(), manifestWorkName, metav1.DeleteOptions{GracePeriodSeconds: &zero})
			if err != nil {
				logs.Logger.Printf("Error deleting ManifestWork after timeout: %v\n", err)
			} else {
				logs.Logger.Printf("Successfully deleted ManifestWork %s after timeout.\n", manifestWorkName)
			}
			return
		}
	}
}

// generateNamespaceManifest generates a namespace manifest for the given namespace.
func generateNamespaceManifest(namespace string) (workv1.Manifest, error) {
	yamlTemplate := `apiVersion: v1
	kind: Namespace
	metadata:
 	name: {{ .NamespaceName }}`

	bodyStringTrimmed := strings.ReplaceAll(yamlTemplate, "\t", "")
	tmpl, err := template.New("namespace").Parse(bodyStringTrimmed)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing template: %v", err)
	}

	namespacePayload := struct {
		NamespaceName string
	}{
		NamespaceName: namespace,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, namespacePayload); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error executing template: %v", err)
	}

	var manifest workv1.Manifest
	if err := yaml.Unmarshal(buf.Bytes(), &manifest); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error unmarshaling manifest: %v", err)
	}
	return manifest, nil
}

// generateServiceAccountManifest creates a ServiceAccount manifest
func generateServiceAccountManifest(namespace, saName string) (workv1.Manifest, error) {
	yamlTemplate := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ .ServiceAccountName }}
  namespace: {{ .Namespace }}`

	tmpl, err := template.New("serviceAccount").Parse(yamlTemplate)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing ServiceAccount template: %v", err)
	}

	manifestPayload := struct {
		ServiceAccountName string
		Namespace          string
	}{
		ServiceAccountName: saName,
		Namespace:          namespace,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, manifestPayload); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error executing ServiceAccount template: %v", err)
	}

	var manifest workv1.Manifest
	if err := yaml.Unmarshal(buf.Bytes(), &manifest); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error unmarshaling ServiceAccount manifest: %v", err)
	}

	return manifest, nil
}

// generateRoleManifest creates a Role manifest with permissions to exec into pods
func generateRoleManifest(namespace, roleName string) (workv1.Manifest, error) {
	yamlTemplate := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: {{ .RoleName }}
  namespace: {{ .Namespace }}
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["pods/exec"]
  verbs: ["create", "get"]`

	tmpl, err := template.New("role").Parse(yamlTemplate)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing Role template: %v", err)
	}

	manifestPayload := struct {
		RoleName  string
		Namespace string
	}{
		RoleName:  roleName,
		Namespace: namespace,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, manifestPayload); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error executing Role template: %v", err)
	}

	var manifest workv1.Manifest
	if err := yaml.Unmarshal(buf.Bytes(), &manifest); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error unmarshaling Role manifest: %v", err)
	}

	return manifest, nil
}

// generateRoleBindingManifest creates a RoleBinding manifest linking the ServiceAccount to the Role
func generateRoleBindingManifest(namespace, roleBindingName, roleName, saName string) (workv1.Manifest, error) {
	yamlTemplate := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: {{ .RoleBindingName }}
  namespace: {{ .Namespace }}
subjects:
- kind: ServiceAccount
  name: {{ .ServiceAccountName }}
  namespace: {{ .Namespace }}
roleRef:
  kind: Role
  name: {{ .RoleName }}
  apiGroup: rbac.authorization.k8s.io`

	tmpl, err := template.New("roleBinding").Parse(yamlTemplate)
	if err != nil {
		return workv1.Manifest{}, fmt.Errorf("error parsing RoleBinding template: %v", err)
	}

	manifestPayload := struct {
		RoleBindingName    string
		RoleName           string
		ServiceAccountName string
		Namespace          string
	}{
		RoleBindingName:    roleBindingName,
		RoleName:           roleName,
		ServiceAccountName: saName,
		Namespace:          namespace,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, manifestPayload); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error executing RoleBinding template: %v", err)
	}

	var manifest workv1.Manifest
	if err := yaml.Unmarshal(buf.Bytes(), &manifest); err != nil {
		return workv1.Manifest{}, fmt.Errorf("error unmarshaling RoleBinding manifest: %v", err)
	}

	return manifest, nil
}

// GetJobStatusFromManifestWork extracts the Job's status from the ManifestWork's status.
func GetJobStatusFromManifestWork(manifestWork *workv1.ManifestWork) (*KubernetesJobStatus, error) {
	if manifestWork == nil {
		return nil, fmt.Errorf("manifestWork is nil")
	}

	for _, manifestCondition := range manifestWork.Status.ResourceStatus.Manifests {
		jobStatus := &KubernetesJobStatus{}
		// Check if the manifest corresponds to a Kubernetes Job
		if manifestCondition.ResourceMeta.Group == "batch" && manifestCondition.ResourceMeta.Resource == "jobs" {

			for _, feedbackValue := range manifestCondition.StatusFeedbacks.Values {
				switch feedbackValue.Name {
				case "JobSucceeded":
					if feedbackValue.Value.Integer != nil {
						jobStatus.JobSucceeded = int(*feedbackValue.Value.Integer)
					} else {
						return nil, fmt.Errorf("expected integer value for JobSucceeded, got nil")
					}
				case "JobFailed":
					if feedbackValue.Value.Integer != nil {
						jobStatus.JobFailed = int(*feedbackValue.Value.Integer)
					} else {
						return nil, fmt.Errorf("expected integer value for JobFailed, got nil")
					}
				}
			}
			return jobStatus, nil
		}
	}

	return nil, fmt.Errorf("Job resource not found in ManifestWork status")
}

// updateNamespaceAndAnnotations updates the namespace and annotations of a Manifest Work object.
func updateNamespaceAndAnnotations(obj runtime.Object, namespace, componentName, jobGroupID string) {

	metaObj, err := meta.Accessor(obj)
	if err != nil {
		logs.Logger.Fatalf("Failed to get metadata accessor: %v", err)
	}
	metaObj.SetNamespace(namespace)

	annotations := metaObj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations["app.cognifog.eu/component"] = componentName
	annotations["app.cognifog.eu/instance"] = jobGroupID
	metaObj.SetAnnotations(annotations)
}

// Deployment Attribute Updates
// ------------------------------------------------

func updateReplicaCount(obj runtime.Object, subType RemediationType) (*workv1.Manifest, error) {
	var replicaNumber int32

	// Handle Deployments
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		replicaNumber = *deployment.Spec.Replicas
		horizontalPodAutoscaling(subType, &replicaNumber)
		deployment.Spec.Replicas = &replicaNumber

		// Create the updated manifest
		rawExtension := runtime.RawExtension{Object: deployment}
		return &workv1.Manifest{RawExtension: rawExtension}, nil
	}

	// Handle StatefulSets
	if statefulSet, ok := obj.(*appsv1.StatefulSet); ok {
		replicaNumber = *statefulSet.Spec.Replicas
		horizontalPodAutoscaling(subType, &replicaNumber)
		statefulSet.Spec.Replicas = &replicaNumber

		// Create the updated manifest
		rawExtension := runtime.RawExtension{Object: statefulSet}
		return &workv1.Manifest{RawExtension: rawExtension}, nil
	}

	// Handle other kinds of objects with replicas here if necessary

	// If the object is not a recognized type with replicas, return nil
	return nil, nil
}

// updateResourceRequirements updates the resource requirements of the deployment based on the remediation type.
func updateResourceRequirements(obj runtime.Object, subType RemediationType) (*workv1.Manifest, error) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, nil
	}

	resources := deployment.Spec.Template.Spec.Containers[0].Resources
	logs.Logger.Print("------Inside updateResourceRequirements----------")
	logs.Logger.Printf("Current CPU: %v, Memory: %v\n", resources.Requests[corev1.ResourceCPU], resources.Requests[corev1.ResourceMemory])
	verticalPodAutoscaling(subType, &resources)
	deployment.Spec.Template.Spec.Containers[0].Resources = resources

	rawExtension := runtime.RawExtension{Object: obj}
	return &workv1.Manifest{RawExtension: rawExtension}, nil
}

// horizontalPodAutoscaling adjusts the replica count for scale up or scale down operations.
func horizontalPodAutoscaling(subType RemediationType, replicas *int32) {
	switch subType {
	case ScaleUp:
		*replicas++
	case ScaleDown:
		*replicas--
	}
}

// verticalPodAutoscaling adjusts the resource requirements for scale out or scale in operations.
func verticalPodAutoscaling(subType RemediationType, resources *corev1.ResourceRequirements) {
	// who is responsible for specifying the resource amount?
	cpuAdjustment := int64(1000)                  // in millicores
	memoryAdjustment := int64(1000 * 1024 * 1024) // in bytes (1000 MiB)
	switch subType {
	case ScaleOut:
		adjustResource(corev1.ResourceCPU, resources, cpuAdjustment)
		adjustResource(corev1.ResourceMemory, resources, memoryAdjustment)
	case ScaleIn:
		adjustResource(corev1.ResourceCPU, resources, -cpuAdjustment)
		adjustResource(corev1.ResourceMemory, resources, -memoryAdjustment)
	}
}

// adjustResource adjusts the resource quantity based on the resource name and adjustment value.
func adjustResource(resourceName corev1.ResourceName, resources *corev1.ResourceRequirements, adjustment int64) {
	currentQuantity := resources.Requests[resourceName]
	newQuantity := currentQuantity.DeepCopy()

	if resourceName == corev1.ResourceCPU {
		newQuantity.Add(*resource.NewMilliQuantity(adjustment, resource.DecimalSI))
	} else if resourceName == corev1.ResourceMemory {
		newQuantity.Add(*resource.NewQuantity(adjustment, resource.BinarySI))
	}

	if newQuantity.Sign() >= 0 {
		resources.Requests[resourceName] = newQuantity
	}
}

// Utility Functions
// ------------------------------------------------

// decodeYAMLToObject decodes a YAML string into a runtime object.
func decodeYAMLToObject(yamlString string) (runtime.Object, error) {
	scheme := runtime.NewScheme()
	appsv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	batchv1.AddToScheme(scheme)

	codecFactory := serializer.NewCodecFactory(scheme)
	decoder := codecFactory.UniversalDeserializer()

	obj, _, err := decoder.Decode([]byte(yamlString), nil, nil)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

// logErrorAndSetJobState logs an error message and sets the job state to the specified state.
func logErrorAndSetJobState(message string, j *Job, state JobState) {
	logs.Logger.Println(message)
	j.State = state
}

// getJobTypeString returns the string representation of a job type.
func getJobTypeString(jobType JobType) string {
	jobString, exists := JobTypeToString[jobType]
	if !exists {
		return "unknownJobType"
	}
	return jobString
}

// FetchNodeUID fetches the UID of a ClusterManager CRD in the hub cluster based on the node name.
func FetchClusterManagerUID(clusterManagerOperatorName string) (string, error) {
	cm, err := clientOperator.ClusterManagers().Get(context.TODO(), "cluster-manager", metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	uuid, err := uuid.Parse(string(cm.UID))
	if err != nil {
		return "", err
	}

	return uuid.String(), nil
}

// Job Struct Methods
// ------------------------------------------------

func (j *Job) PromoteJob(authHeader string, OwnerID string) error {
	logs.Logger.Println("promoting job...")

	// Create the JSON payload
	payload := struct {
		OwnerID string `json:"owner_id"`
	}{
		OwnerID: OwnerID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logs.Logger.Println("Error marshaling JSON payload:", err)
		return err
	}

	// Create the request with JSON payload
	reqState, err := http.NewRequest("PATCH", jobmanagerBaseURL+"jobmanager/jobs/promote/"+j.ID, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logs.Logger.Println("Error creating Job locking request:", err)
		return err
	}
	reqState.Header.Add("Authorization", authHeader)
	reqState.Header.Add("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(reqState)
	if err != nil {
		logs.Logger.Println("Error occurred during Job locking request:", err)
		return err
	}
	defer resp.Body.Close()

	logs.Logger.Println("GET Lock Response", resp.Status)
	return nil
}

func (j *Job) StateMapper(state workv1.ManifestWorkStatus) {
	offset := len(state.Conditions)
	if offset >= 1 {
		offset = len(state.Conditions) - 1
	}
	switch jobState := state.Conditions[offset].Type; jobState {
	case "Progressing":
		j.State = JobState("Progressing")
	case "Available", "Applied":
		j.State = JobState("Finished")
	case "Degraded":
		j.State = JobState("Degraded")
	default:
		j.State = JobState("Progressing")
	}
}

func (j *Job) UpdateJobResource(manifestWork *workv1.ManifestWork) {
	if manifestWork != nil {
		if len(manifestWork.Status.Conditions) != 0 {
			j.StateMapper(manifestWork.Status)
		} else {
			// Let's assume that the job is still progressing
			j.State = Progressing
		}
		j.Resource.ResourceUID = string(manifestWork.UID)
		j.Resource.ResourceName = manifestWork.Name
		j.Resource.Conditions = append(j.Resource.Conditions, manifestWork.Status.Conditions...)
	} else {
		j.Resource.ResourceName = ""
		j.Resource.Conditions = append(j.Resource.Conditions, metav1.Condition{Type: "Applied", Status: "True", ObservedGeneration: 0, Reason: "Deleted", Message: "Resource has been deleted"})
	}
	logs.Logger.Printf("Job's Resource details: %#v", j.Resource)
}
