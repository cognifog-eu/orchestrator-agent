package models

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	workv1 "open-cluster-management.io/api/work/v1"
)

var (
	jobmanagerBaseURL  = os.Getenv("JOBMANAGER_URL") // "http://jobmanager-service:8082"
	lighthouseBaseURL  = os.Getenv("LIGHTHOUSE_BASE_URL")
	apiV3              = "/api/v3"
	matchmackerBaseURL = os.Getenv("MATCHMAKING_URL")
)

type Resource struct {
	ID           uuid.UUID `json:"id"`
	ManifestName string    `json:"resource_name"`
	NodeTarget   string    `json:"node_target"`
	Status       Status    `json:"status"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Status struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func CreateManifestWork(target Target, manifestWorkYaml string) (string, error) {
	name := "deploy-test-"
	namespace := target.ClusterName
	// TODO validate if work doesnt exist already
	if ExistsManifestWork(namespace, name) {
		fmt.Println("ManifestWork " + name + " already exists!")
		return "", errors.New("ManifestWork " + name + " already exists!") //message
	}
	// fmt.Println(work.Spec.Workload.Manifests)

	var manifestWork *workv1.ManifestWork
	// decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()
	decoder := scheme.Codecs.UniversalDeserializer()
	manifestWork = &workv1.ManifestWork{}
	err := runtime.DecodeInto(decoder, []byte(manifestWorkYaml), manifestWork)
	if err != nil {
		fmt.Println(err)
		// panic(err)
	}
	fmt.Println("Sending manifest to OCM...")
	manifestWork, err = clientsetWorkOper.WorkV1().ManifestWorks(target.ClusterName).
		Create(context.TODO(), manifestWork, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("ERROR: ", err)
		// panic(err)
		return "", errors.New(err.Error())
	}
	return string(manifestWork.GetUID()), err
}

func GetManifestWork(namespace string, manifestWorkName string) (*workv1.ManifestWork, error) {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	manifest, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).
		Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	// log.Debug("ExistsManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil)) //err.Error())
	if err == nil {
		fmt.Println("ERROR: ", err)
	}
	return manifest, err
}

func CheckStatusManifestWork(namespace string, manifestWorkName string) *workv1.ManifestWorkStatus {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	// log.Debug("Obtaining status... ") //err.Error())
	result, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).
		Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error obtaining ManifestWork status")
	}
	//	setManifestWorkCache(namespace, manifestWorkName)
	//}
	return &result.Status // TODO update to be an array
}

func ExistsManifestWork(namespace string, manifestWorkName string) bool {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	_, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	// log.Debug("ExistsManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil)) //err.Error())
	//if err == nil {
	//	setManifestWorkCache(namespace, manifestWorkName)
	//}
	return err == nil
}

func DeleteManifestWork(namespace string, manifestWorkName string) bool {
	if !ExistsManifestWork(namespace, manifestWorkName) {
		// log.Debug("ServiceMonitor " + manifestWorkName + " does not exist!")
		return false
	}
	//err := clientset.CoreV1().Services(namespace).Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
	err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).Delete(context.TODO(), manifestWorkName, metav1.DeleteOptions{})
	// log.Debug("DeleteManifestWork: " + manifestWorkName + " " + strconv.FormatBool(err == nil))
	return err == nil
}

func ListManifestWork(namespace string) *workv1.ManifestWorkList {
	manifestlist, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		// log.Debug("Error obtaining ManifestWorkList")
	}
	return manifestlist
}

func ResourceSync() ([]Resource, error) {
	var err error
	var resources []Resource
	// get all manifestworks
	var manifestStatus *workv1.ManifestWorkStatus
	// var managedClusters clusterv1.ManagedClusterList
	managedClusters, err := clientsetClusterOper.ClusterV1().ManagedClusters().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Println("Error obtaining managed clusters")
	}
	for _, managedCluster := range managedClusters.Items {
		allManifestWorks := ListManifestWork(managedCluster.Name)
		// for each manifestwork
		if len(allManifestWorks.Items) > 0 {
			for _, manifestWork := range allManifestWorks.Items {
				// get status
				manifestStatus = &manifestWork.Status
				// find job with the corresponding UID, should I assume it exists?
				manifestUID := uuid.MustParse(string(string(manifestWork.UID)))
				resource := Resource{
					ID:           manifestUID,
					ManifestName: manifestWork.Name,
					Status: Status{
						manifestStatus.Conditions,
					},
				}
				resources = append(resources, resource)
			}
		} else {
			fmt.Println("No Resources were found during sync up process for cluster: " + managedCluster.Name)
		}
	}
	return resources, err
}
