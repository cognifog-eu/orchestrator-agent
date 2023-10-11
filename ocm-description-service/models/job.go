package models

import (
	"flag"
	"path/filepath"

	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ID       uuid.UUID       `json:"id"`
	UUID     uuid.UUID       `json:"uuid"` // unique across all ICOS
	Type     JobType         `json:"type"`
	State    State           `json:"state"`
	JobGroup JobGroup        `json:"group"`
	Manifest workv1.Manifest `json:"manifest"` // will be an array in the future
	Targets  []Target        // array of targets where the manifest is applied
	Locker   bool            `json:"locker"`
	// Policies?
	// Requirements?
}

type Target struct {
	ID       uint32 `json:"id"`
	Hostname string `json:"hostname"`
	// UPC to define
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
	//ServiceAPIPath string			// POST /api/v1/namespaces/{namespace}/services
	//ServiceMonitorAPIPath string	// POST /api/v1/namespaces/{namespace}/
}

func InClusterConfig() error {
	config, err := rest.InClusterConfig()

	// Outside of the cluster for development
	if err != nil {
		//panic(err.Error())
		var kubeconfig *string
		// log.Debug("The home folder is: ", homedir.HomeDir())
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config-files/config.portainer.yaml"), "(optional) absolute path to the kubeconfig file")
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
	// take unmarshalled job, convert it to manifest work
	work := workv1.ManifestWork{
		TypeMeta: v1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      j.JobGroup.AppName,
			Namespace: "",
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					j.Manifest,
				},
			},
		},
	}

	// map each target to ManifestWork.metadata.namespace
	// offload job to Orchestrator in-cluster
	// creates the in-cluster config
	err = InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// var uid string

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

	// IT1 ONLY
	work.Namespace = j.Targets[0].Hostname
	// retrieve the uuid and status of the job from OCM
	j.UUID = uuid.MustParse(CreateManifestWork(j.Targets[0], &work))
	j.StateMapper(CheckStatusManifestWork(j.Targets[0].Hostname, work.Name))
	// lock the job so other instance won't take it
	j.Locker = true

	// return status, this should be a map[uid,state:target]
	return err
}

func (j *Job) StateMapper(state string) {
	switch jobState := state; jobState {
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
