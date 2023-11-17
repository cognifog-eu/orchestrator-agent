package models

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	workv1 "open-cluster-management.io/api/work/v1"
)

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
