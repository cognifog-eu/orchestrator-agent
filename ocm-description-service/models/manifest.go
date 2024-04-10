/*
Copyright 2023 Bull SAS

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

type ManifestWork interface{}
type Manifests []Manifest

type Manifest struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	Spec       Spec     `json:"spec"`
}

type ManifestMappers []ManifestMapper

type ManifestMapper struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	// Namespace string `yaml:"-"` // TODO some manifest are not namespaced
}

type Spec struct {
	Selector Selector `json:"selector"`
	Template Template `json:"template"`
}

type Selector struct {
	MatchLabels Labels `json:"matchLabels"`
}

type Labels struct {
	App string `json:"app"`
}

type Template struct {
	Metadata Metadata     `json:"metadata"`
	Spec     TemplateSpec `json:"spec"`
}

type ObjMetadata struct {
	Labels Labels `json:"labels"`
}

type TemplateSpec struct {
	Containers []Container `json:"containers"`
}

type Container struct {
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	Command []string `json:"command"`
}
