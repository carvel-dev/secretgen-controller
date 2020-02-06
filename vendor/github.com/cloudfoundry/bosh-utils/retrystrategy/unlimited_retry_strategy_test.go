package retrystrategy_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	. "github.com/cloudfoundry/bosh-utils/retrystrategy"
)

var _ = Describe("UnlimitedRetryStrategy", func() {
	var (
		logger boshlog.Logger
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	Describe("Try", func() {
		It("stops retrying when it receives a non-retryable error", func() {
			output := []attemptOutput{}
			for i := 0; i < 6; i++ {
				output = append(output, attemptOutput{
					ShouldRetry: true,
					AttemptErr:  fmt.Errorf("error-%d", i),
				})
			}
			output = append(output, attemptOutput{
				ShouldRetry: false,
				AttemptErr:  errors.New("final-error"),
			})
			retryable := newSimpleRetryable(output)

			attemptRetryStrategy := NewUnlimitedRetryStrategy(0, retryable, logger)
			err := attemptRetryStrategy.Try()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("final-error"))
			Expect(retryable.Attempts).To(Equal(7))
		})

		It("stops retrying when it stops receiving errors", func() {
			output := []attemptOutput{}
			for i := 0; i < 6; i++ {
				output = append(output, attemptOutput{
					ShouldRetry: true,
					AttemptErr:  fmt.Errorf("error-%d", i),
				})
			}
			output = append(output, attemptOutput{
				ShouldRetry: false,
				AttemptErr:  nil,
			})

			retryable := newSimpleRetryable(output)

			attemptRetryStrategy := NewUnlimitedRetryStrategy(0, retryable, logger)
			err := attemptRetryStrategy.Try()
			Expect(err).ToNot(HaveOccurred())
			Expect(retryable.Attempts).To(Equal(7))
		})
	})
})
