package models

import (
	"context"
	"flag"
	"fmt"
	"icos/server/ocm-description-service/utils/logs"
	"path/filepath"
	"time"
    yaml "gopkg.in/yaml.v2"
	"github.com/google/uuid"
	y "gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	workv1 "open-cluster-management.io/api/work/v1"

	// open-cluster-management
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	workclient "open-cluster-management.io/api/client/work/clientset/versioned"
)

var clientset *kubernetes.Clientset
var clientsetWorkOper workclient.Interface
var clientsetClusterOper clusterclient.Interface

type State int
type JobType int
type OrchestratorType string

const (
	OCM OrchestratorType = "ocm"
)

const (
	// Valid condition types are:
	// 1. Applied represents workload in ManifestWork is applied successfully on managed cluster.
	// 2. Progressing represents workload in ManifestWork is being applied on managed cluster.
	// 3. Available represents workload in ManifestWork exists on the managed cluster.
	// 4. Degraded represents the current state of workload does not match the desired
	Applied State = iota + 1
	Progressing
	Available
	Degraded

	CreateDeployment JobType = iota + 1
	GetDeployment
	DeleteDeployment
	RecoveryJob
	CreateNamespace

	ScaleIn
	ScaleOut
)

var JobTypeFromString = map[string]JobType{
	"CreateDeployment": CreateDeployment,
	"GetDeployment":    GetDeployment,
	"DeleteDeployment": DeleteDeployment,
	"RecoveryJob":      RecoveryJob,
	"CreateNamespace":  CreateNamespace,
	"ScaleIn":          ScaleIn,
	"ScaleOut":         ScaleOut,
}

type Job struct {
	ID           uuid.UUID        `json:"id"`
	UUID         uuid.UUID        `json:"uuid"` // unique across all ICOS, represents resource UUID
	Type         JobType          `json:"type,omitempty"`
	State        State            `json:"state"`
	JobGroup     JobGroup         `json:"group,omitempty"`
	Manifest     string           `json:"manifest"` // represents manifests to be applied, will be an array in the future
	Manifests    []string         `json:"manifests,omitempty"`
	Targets      []Target         // array of targets where the manifest is applied
	Orchestrator OrchestratorType `gorm:"type:text" json:"orchestrator"` // identifies the orchestrator that can execute the job based on target provided by MM
	Locker       *bool            `json:"locker"`
	UpdatedAt    time.Time        `json:"updatedAt"`
	Resource     Resource         `json:"resource"`
	Namespace    string           `json:"namespace"`
	// Policies?
	// Requirements?
}

func JobTypeIsRecoveryAction(value int) bool {
	return int(ScaleIn) <= value && value >= int(ScaleOut)
}

func JobTypeIsCreateNamespace(value int) bool {
	return int(CreateNamespace) == value
}

type Target struct {
	ID          uint32
	JobID       uuid.UUID
	ClusterName string `json:"cluster_name"`
	NodeName    string `json:"node_name"`
	// what we need to know about peripherals
	// TODO UPC&AGGREGATOR
}

// hold information that N jobs share (N jobs needed to provide application x)
type JobGroup struct {
	AppInstanceID  uuid.UUID `json:"job_group_id"`
	AppName        string    `json:"job_group_name"`
	AppDescription string    `json:"job_group_description"`
}

type KubeConfig struct {
	Name      string // Name of the cluster
	Server    string // https://{ip}:{port}
	Namespace string // "core"
	User      string // service account
	Token     string // pass
}

func InClusterConfig() error {
	config, err := rest.InClusterConfig()

	// Outside of the cluster for development
	if err != nil {
		//panic(err.Error())
		var kubeconfig *string
		logs.Logger.Println("The home folder is: ", homedir.HomeDir())
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "C:\\Users\\a880237\\.kube\\config")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
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

	return err
}

func Execute(j *Job) (*Job, error) {
	var err error
	resUUID := j.UUID.String()
	// namespace work creation
	if JobTypeIsCreateNamespace(int(j.Type)) {
		logs.Logger.Println("Creating Work for NS Job: " + j.ID.String())
		manifestWorkNS := CreateNSWork(j)
		// send ManifestWork to OCM API Server
		_, err = clientsetWorkOper.WorkV1().ManifestWorks(j.Targets[0].ClusterName).Create(context.TODO(), manifestWorkNS, metav1.CreateOptions{})
		if err != nil {
			// if error, manifeswork not created!
			// panic(err.Error())
			j.Resource.Conditions = append(j.Resource.Conditions,
				metav1.Condition{
					Type:    workv1.WorkDegraded,
					Status:  metav1.StatusFailure,
					Message: "manifestworks.work.open-cluster-management.io" + j.Resource.ManifestName + "already exists",
				})
			logs.Logger.Println("error occured: ", err)
		}
	}
	// if execution requires the creation of a new manifest work
	if !JobTypeIsRecoveryAction(int(j.Type)) {
		logs.Logger.Println("Creating Work for Job: " + j.ID.String())
		// create valid ManifestWork object
		manifestWork := CreateWork(j)
		// send ManifestWork to OCM API Server
		manifestWork, err = clientsetWorkOper.WorkV1().ManifestWorks(j.Targets[0].ClusterName).Create(context.TODO(), manifestWork, metav1.CreateOptions{})
		if err != nil {
			// if error, manifeswork not created!
			// panic(err.Error())
			j.Resource.Conditions = append(j.Resource.Conditions,
				metav1.Condition{
					Type:    workv1.WorkDegraded,
					Status:  metav1.StatusFailure,
					Message: "manifestworks.work.open-cluster-management.io" + j.Resource.ManifestName + "already exists",
				})
			logs.Logger.Println("error occured: ", err)
		} // retrieve the UID created by OCM
		resUUID = string(manifestWork.GetUID())
	}
	// retrieve the uuid and status of the applied manifest from OCM
	if resUUID != "" {
		// temporal sleep to ensure status is obtained during deploy
		time.Sleep(time.Second * 10)
		appliedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Targets[0].ClusterName).Get(context.TODO(), j.Resource.ManifestName, metav1.GetOptions{})
		logs.Logger.Println(appliedManifestWork.Name)
		// skip if errors and set to Degraded
		if err != nil || appliedManifestWork == nil {
			logs.Logger.Println("Error obtaining applied ManifestWork status")
			j.State = Degraded
		} else {
			logs.Logger.Println("manifestWork uid: ", resUUID)
			// TODO improve validation mustParse panics!
			j.UUID = uuid.MustParse(resUUID)
			// if the job is a recovery action
			if JobTypeIsRecoveryAction(int(j.Type)) { // TODO refactor
				appliedManifestWork, err = PatchWork(j)
				if err != nil {
					logs.Logger.Println("Error Patching ManifestWork")
					j.State = Degraded
				}
			}
			// if conditions empty skip state mapper,
			if len(appliedManifestWork.Status.Conditions) != 0 {
				j.StateMapper(appliedManifestWork.Status)
			} else {
				// assumption: manifestwork is in progress
				j.State = Progressing
			}
			// lock the job so other instance won't take it -> TODO: race condition is still possible!
			b := new(bool)
			*b = true
			j.Locker = b
			j.Resource.ID = j.UUID
			j.Resource.ManifestName = appliedManifestWork.Name
			// populate conditions slice
			j.Resource.Conditions = append(j.Resource.Conditions, appliedManifestWork.Status.Conditions...)
			fmt.Printf("Job's Resource details: %#v", j)
		}
	} else {
		logs.Logger.Println("Manifest UID could not be retrieved")
	}
	// return status, this should be a map[uid,state:target]
	return j, err
}

func (j *Job) ManifestMapper(manifest string) {
	j.Manifests = append(j.Manifests, manifest)
}

func (j *Job) StateMapper(state workv1.ManifestWorkStatus) {
	offset := len(state.Conditions)
	if len(state.Conditions) > 1 {
		offset = len(state.Conditions) - 1
	}
	switch jobState := state.Conditions[offset].Type; jobState {
	case "Progressing":
		j.State = Progressing
	case "Available":
		j.State = Available
	case "Degraded":
		j.State = Degraded
	default:
		j.State = Applied
	}
}

func CreateNSWork(j *Job) *workv1.ManifestWork {
	var manifest *workv1.Manifest
	var err error
	// namespace creation work
	// create manifest then marshal it
	nSManifests := ManifestMappers{}
	nSManifest := ManifestMapper{
		APIVersion: "v1",
		Kind:       "Namespace",
		Metadata: Metadata{
			Name: j.Namespace,
		},
	}
	nSManifests = append(nSManifests, nSManifest)
	logs.Logger.Printf("%#v", nSManifests)
	// nSManifestBytes := []byte(fmt.Sprintf("%#v", nSManifest))
	nSManifestBytes, err := y.Marshal(&nSManifests)
	if err != nil {
		logs.Logger.Println("Could not marshal namespace manifest" + err.Error())
	}
	err = y.Unmarshal(nSManifestBytes, &manifest)
	logs.Logger.Printf("%#v", manifest)
	if err != nil {
		logs.Logger.Println("Could not unmarshal namespace manifest" + err.Error())
	}
	fmt.Printf("Manifest details: %#v", nSManifestBytes)
	workNS := workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: j.JobGroup.AppName,
			// GenerateName: "deploy-app-", // TODO change
			Namespace: j.Targets[0].ClusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					*manifest,
				},
			},
		},
	}
	return &workNS
}

func CreateWork(j *Job) *workv1.ManifestWork {
	var manifest *workv1.Manifest
	yaml.Unmarshal([]byte(j.Manifest), &manifest)
	// var err error

	// yaml.Unmarshal([]byte(j.Manifest), &manifest)
	// // ensure namespace exists TODO
	// var manifestMapper Manifest
	// json.Unmarshal(manifest.Raw, &manifestMapper)
	// fmt.Printf("Uncoded manifest: %#v", manifestMapper)
	// manifestMapper.Namespace = j.Namespace
	// manifestBodyBytes, err := json.Marshal(manifestMapper)
	// if err != nil {
	// 	logs.Logger.Println("Error adding Namespace to Manifest")
	// 	j.State = Degraded
	// }
	// manifest.Raw = manifestBodyBytes // TODO test

	work := workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"app.icos.eu/name":      j.JobGroup.AppName,
				"app.icos.eu/component": j.Resource.ManifestName,
				"app.icos.eu/instance":  j.JobGroup.AppInstanceID.String(),
			},
			Name: j.Resource.ManifestName,
			// GenerateName: "deploy-app-", // TODO test
			Namespace: j.Targets[0].ClusterName,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					*manifest,
				},
			},
		},
	}

	return &work
}

func PatchWork(j *Job) (*workv1.ManifestWork, error) {
	appliedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Targets[0].ClusterName).Patch(context.TODO(), j.Resource.ManifestName, types.StrategicMergePatchType, []byte(j.Manifest), metav1.PatchOptions{})
	return appliedManifestWork, err
}
