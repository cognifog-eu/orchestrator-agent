package models

type Manifests []Manifest

type Manifest struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace"`
	// Spec       Spec        `json:"spec"`
}

type Metadata struct {
	Name string `json:"name"`
}

// type Spec struct {
// 	Selector Selector `json:"selector"`
// 	Template Template `json:"template"`
// }

// type Selector struct {
// 	MatchLabels Labels `json:"matchLabels"`
// }

// type Labels struct {
// 	App string `json:"app"`
// }

// type Template struct {
// 	Metadata Metadata     `json:"metadata"`
// 	Spec     TemplateSpec `json:"spec"`
// }

// type Metadata struct {
// 	Labels Labels `json:"labels"`
// }

// type TemplateSpec struct {
// 	Containers []Container `json:"containers"`
// }

// type Container struct {
// 	Name    string   `json:"name"`
// 	Image   string   `json:"image"`
// 	Command []string `json:"command"`
// }
