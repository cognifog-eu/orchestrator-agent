package formaterror

import (
	"errors"
	"strings"
)

func FormatError(err string) error {

	if strings.Contains(err, "name") {
		return errors.New("Name Already Taken")
	}
	return errors.New("Incorrect Details")
}
