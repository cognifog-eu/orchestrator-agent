package models

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	workv1 "open-cluster-management.io/api/work/v1"
)

// type ManifestWork struct {
// 	ApiVersion string
// 	Kind       string
// 	Metadata   Metadata
// 	WorkSpec   WorkSpec
// }

type Metadata struct {
	Namespace string
	Name      string
}

type WorkSpec struct {
	Workload Workload
}

type Workload struct {
	Manifest Manifest // TODO will be an array
}

type Manifest struct {
	workv1.Manifest
	ApiVersion string
	Kind       string
	Metadata   Metadata
	Spec       Spec
}

type Spec struct {
	Selector Selector
	Template Template
}

type Selector struct {
	MatchLabels map[string]string `yaml:"MatchLabels"`
}

type Template struct {
	TemplateMetadata TemplateMetadata
	TemplateSpec     TemplateSpec
}

type TemplateMetadata struct {
	Labels map[string]string `yaml:"labels"`
}

type TemplateSpec struct {
	Containers []Container
}

type Container struct {
	Name      string    `yaml:"name"`
	Image     string    `yaml:"image"`
	Command   []string  `yaml:"command"`
	Args      []string  `yaml:"args"`
	Resources Resources `yaml:"resources"`
}

type Resources struct {
	Requests map[string]string `yaml:"requests"`
	Limits   map[string]string `yaml:"limits"`
}

func CreateManifestWork(target Target, work *workv1.ManifestWork) string {

	// TODO validate if work doesnt exist already
	if ExistsManifestWork(work.Namespace, work.Name) {
		// log.Debug("ManifestWork " + manifestWorkName + " already exists!")
		// message := "ManifestWork already exists"
		return "" //message
	}
	// log.Debug(manifest)

	fmt.Println("Sending manifest to OCM...")
	manifestWork, err := clientsetWorkOper.WorkV1().ManifestWorks(target.NodeName).Create(context.TODO(), work, metav1.CreateOptions{})
	if err != nil {
		// debug
		fmt.Println(manifestWork, err)
		// panic(err)
		return err.Error()
	}
	// log.Debug(manifestWork.GetSelfLink())
	return string(manifestWork.GetUID())
}

func CheckStatusManifestWork(namespace string, manifestWorkName string) string {
	//if getManifestWorkCache(namespace, manifestWorkName) {
	//	return true
	//}
	//_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
	// log.Debug("Obtaining status... ") //err.Error())
	result, err := clientsetWorkOper.WorkV1().ManifestWorks(namespace).
		Get(context.TODO(), manifestWorkName, metav1.GetOptions{})
	if err != nil {
		// log.Debug("Error obtaining ManifestWork status")
	}
	//	setManifestWorkCache(namespace, manifestWorkName)
	//}
	return result.Status.Conditions[0].Type // TODO update to be an array
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
