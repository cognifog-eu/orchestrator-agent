package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
	UUID           uuid.UUID      `json:"uuid"`
	Type           JobType        `json:"type"`
	State          State          `json:"state"`
	AppDescription AppDescription `json:"component"` // will be an array in the future
	Targets        []Target       // array of targets where the AppDescription is applied
	// Policies?
	// Requirements?
}
type Target struct {
}

type AppDescription struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas int `yaml:"replicas"`
		Selector struct {
			MatchLabels struct {
				Env string `yaml:"env"`
			} `yaml:"matchLabels"`
		} `yaml:"selector"`
		Template struct {
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
			Spec struct {
				Containers []struct {
					Name      string   `yaml:"name"`
					Image     string   `yaml:"image"`
					Command   []string `yaml:"command"`
					Args      []string `yaml:"args"`
					Resources struct {
						Requests struct {
						} `yaml:"requests"`
						Limits struct {
						} `yaml:"limits"`
					} `yaml:"resources"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

type Jobs []struct {
	Job Job
}

func (j *Job) SaveJob(db *gorm.DB) (*Job, error) {

	err := db.Debug().Create(&j).Error
	if err != nil {
		return &Job{}, err
	}
	return j, nil
}

func (j *Job) FindJobByUUID(db *gorm.DB, uid uint32) (*Job, error) {
	err := db.Debug().Model(Job{}).Where("id = ?", uid).Take(&j).Error
	if err != nil {
		return &Job{}, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &Job{}, errors.New("Job Not Found")
	}
	return j, err
}

func (j *Job) UpdateAJob(db *gorm.DB, uid uint32) (*Job, error) {
	db = db.Debug().Model(&Job{}).Where("id = ?", uid).Take(&Job{}).UpdateColumns(
		map[string]interface{}{
			"state":      j.State,
			"created_at": time.Now(),
		},
	)
	if db.Error != nil {
		return &Job{}, db.Error
	}

	// This is the display the updated Job
	err := db.Debug().Model(&Job{}).Where("id = ?", uid).Take(&j).Error
	if err != nil {
		return &Job{}, err
	}
	return j, nil
}

func (j *Job) DeleteAJob(db *gorm.DB, uid uint32) (int64, error) {

	// db = db.Debug().Model(&Job{}).Where("id = ?", uid).Take(&Job{}).Delete(&Job{}) // debug only
	db = db.Model(&Job{}).Where("id = ?", uid).Take(&Job{}).Delete(&Job{})

	if db.Error != nil {
		return 0, db.Error
	}
	return db.RowsAffected, nil
}
