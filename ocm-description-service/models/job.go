package models

import (
	"flag"
	"path/filepath"

	"github.com/google/uuid"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	Created State = iota + 1
	Started
	Progressing
	Finished
	Failed

	CreateDeployment JobType = iota + 1
	GetDeployment
	DeleteDeployment
)

type Job struct {
	UUID     uuid.UUID `json:"uuid"`
	Type     JobType   `json:"type"`
	State    State     `json:"state"`
	Manifest Manifest  `json:"manifest"` // will be an array in the future
	Targets  []Target  // array of targets where the manifest is applied
	// Policies?
	// Requirements?
}

type Target struct {
	ID string
	// UPC to define
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

func (j *Job) Execute() (string, error) {
	var err error
	// take unmarshalled job, convert it to manifest work
	work := workv1.ManifestWork{
		TypeMeta: v1.TypeMeta{
			Kind:       "ManifestWork",
			APIVersion: "work.open-cluster-management.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:         j.Manifest.Metadata.Name,
			GenerateName: j.Manifest.Metadata.Name,
			Namespace:    "",
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					workv1.Manifest{
						RawExtension: runtime.RawExtension{},
					},
				},
			},
		},
		// ApiVersion: "work.open-cluster-management.io/v1",
		// Kind:       "ManifestWork",
		// Metadata: Metadata{
		// 	Namespace: "target-cluster-id", // target_id
		// 	Name:      "app-name",
		// },
		// WorkSpec: WorkSpec{
		// 	Workload: Workload{ // the actual workload to pass to OCM
		// 		Manifest: j.Manifest,
		// 	},
		// },
	}

	// map each target to ManifestWork.metadata.namespace
	// offload job to Orchestrator in-cluster
	// creates the in-cluster config
	err = InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	for _, target := range j.Targets {
		work.Metadata.Namespace = target.ID
		CreateManifestWork(target, work)
	}

	// retrieve the uuid and status of the job from OCM
	// sync it with job manager
	// return status
	return "status", err
}
