package services

import (
	"strings"
)

type MultiErrors []error

func (errs MultiErrors) Error() string {
	var r strings.Builder
	for idx, err := range errs {
		if idx > 0 {
			r.WriteString(", ")
		}
		r.WriteString(err.Error())
	}
	return r.String()
}
