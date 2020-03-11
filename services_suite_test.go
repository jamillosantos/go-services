//go:generate go run github.com/golang/mock/mockgen -destination=mocks_test.go -package services_test . Resource,Server,Reporter,Configurable,RetrierReporter
package services_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Services Tests")
}
