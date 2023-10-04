package models

type ManifestWork struct {
	ApiVersion string
	Kind       string
	Metadata   Metadata
	WorkSpec   WorkSpec
}

type Metadata struct {
	Namespace string
	Name      string
}

type WorkSpec struct {
	Workload Workload
}

type Workload struct {
	Manifest []Manifest
}

type Manifest struct {
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

func execute(job Job) {
	// take job, unmarshal it to manifest work
	// map each target to ManifestWork.metadata.namespace
	// offload job to Orchestrator in-cluster
	// retrieve the uuid and status of the job from OCM
	// sync it with job manager
	// return status
}
