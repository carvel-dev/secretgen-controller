package server_test

import (
	. "github.com/cloudfoundry/config-server/server"

	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("errorResponse", func() {
	Context("#toJSON", func() {
		It("should return a JSON with the error string from error struct", func() {
			err := errors.New("whatever")

			response := NewErrorResponse(err).GenerateErrorMsg()

			Expect(response).To(Equal(`{"error":"whatever"}`))
		})
	})
})
