package integration_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	pathToConfigServer string
	pathToConfigFile   string
)

func TestIntegrationTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IntegrationTests Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	configServerPath, err := gexec.Build("github.com/cloudfoundry/config-server")
	Expect(err).NotTo(HaveOccurred())
	return []byte(configServerPath)
}, func(data []byte) {
	pathToConfigServer = string(data)
	pathToConfigFile = os.Getenv("CONFIG_FILE")
})
