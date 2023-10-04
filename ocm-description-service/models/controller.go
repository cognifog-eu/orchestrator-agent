package models

type ICOSController struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	// CreationTimestamp time.Time `json:"creationTimestamp"`
}

type ICOSControllers []struct {
	ICOSController ICOSController
}
