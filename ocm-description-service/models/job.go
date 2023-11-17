package models

import (
	"context"
	"errors"
	"flag"
	"icos/server/ocm-description-service/utils/logs"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	workv1 "open-cluster-management.io/api/work/v1"

	// open-cluster-management
	workclient "open-cluster-management.io/api/client/work/clientset/versioned"
)

var clientset *kubernetes.Clientset
var clientsetWorkOper workclient.Interface

type State int
type JobType int

const (
	// Valid condition types are:
	// 1. Applied represents workload in ManifestWork is applied successfully on managed cluster.
	// 2. Progressing represents workload in ManifestWork is being applied on managed cluster.
	// 3. Available represents workload in ManifestWork exists on the managed cluster.
	// 4. Degraded represents the current state of workload does not match the desired
	Created State = iota + 1
	Progressing
	Available
	Degraded

	CreateDeployment JobType = iota + 1
	GetDeployment
	DeleteDeployment
)

type Job struct {
	ID        uuid.UUID `json:"id"`
	UUID      uuid.UUID `json:"uuid"` // unique across all ICOS
	Type      JobType   `json:"type,omitempty"`
	State     State     `json:"state"`
	JobGroup  JobGroup  `json:"group,omitempty"`
	Manifest  string    `json:"manifest"` // represents manifests to be applied, will be an array in the future
	Manifests []string  `json:"manifests,omitempty"`
	Targets   []Target  // array of targets where the manifest is applied
	Locker    *bool     `json:"locker"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Policies?
	// Requirements?
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
	AppName        string `json:"appName"`
	AppDescription string `json:"appDescription"`
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
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
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

	return err
}

func (j *Job) Execute() error {
	var err error
	if len(j.Targets) > 0 {
		// create the in-cluster config
		err = InClusterConfig()
		if err != nil {
			// panic(err.Error())
		}
		logs.Logger.Println("Creating Work for Job: " + j.ID.String())
		// create valid ManifestWork object
		manifestWork := j.CreateWork()
		// send ManifestWork to OCM API Server
		manifestWork, err = clientsetWorkOper.WorkV1().ManifestWorks(j.Targets[0].ClusterName).Create(context.TODO(), manifestWork, metav1.CreateOptions{})
		// newUUID, err := CreateManifestWork(j.Targets[0], string(work))
		if err != nil {
			// panic(err.Error())
			logs.Logger.Println("error occured: ", err)
		} // retrieve the UID created by OCM
		newUUID := string(manifestWork.GetUID())
		// // retrieve the uuid and status of the job from OCM
		if newUUID != "" {
			// state := CheckStatusManifestWork(j.Targets[0].ClusterName, string(work))
			appliedManifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(j.Targets[0].ClusterName).Get(context.TODO(), manifestWork.Name, metav1.GetOptions{})
			if err != nil {
				logs.Logger.Println("Error obtaining ManifestWork status")
			}
			logs.Logger.Println("manifestWork uid: ", newUUID)

			// NEXT ITERATION
			// take into account that manifest:target is 1 to 1 relationship
			// job should contain N pairs manifest:target
			// for each target create a manifestwork within the target namespace, retrieve its uid and state
			// for _, target := range j.Targets {
			// 	work.Namespace = target.ID
			// 	// retrieve the uuid and status of the job from OCM
			// 	uid := CreateManifestWork(target, &work)
			// 	state := CheckStatusManifestWork(target.ID, work.Name)
			// 	j.StateMapper(state)
			// }

			logs.Logger.Println("Work UUID is: " + newUUID)
			j.UUID = uuid.MustParse(newUUID)
			j.StateMapper(appliedManifestWork.Status)
			// lock the job so other instance won't take it
			b := new(bool)
			*b = true
			j.Locker = b
		}
	} else {
		err = errors.New("Target Cannot be empty")
	}
	// return status, this should be a map[uid,state:target]
	return err
}

func (j *Job) ManifestMapper(manifest string) {
	j.Manifests = append(j.Manifests, manifest)
}

func (j *Job) StateMapper(state workv1.ManifestWorkStatus) {
	switch jobState := state.Conditions[len(state.Conditions)-1].Type; jobState {
	case "Progressing":
		j.State = Progressing
	case "Available":
		j.State = Available
	case "Degraded":
		j.State = Degraded
	default:
		j.State = Created
	}
}

func (j *Job) CreateWork() *workv1.ManifestWork {
	var manifest *workv1.Manifest
	yaml.Unmarshal([]byte(j.Manifest), &manifest)
	work := workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Name:         j.JobGroup.AppName,
			GenerateName: "deploy-app-", // TODO change
			Namespace:    j.Targets[0].ClusterName,
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
