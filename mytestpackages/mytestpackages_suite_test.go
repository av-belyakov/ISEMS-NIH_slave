package mytestpackages_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMytestpackages(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mytestpackages Suite")
}
