package work_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	"github.com/cloudfoundry/bosh-utils/work"
)

var _ = Describe("Pool", func() {
	It("runs the given tasks", func() {
		pool := work.Pool{
			Count: 2,
		}

		resultsChan := make(chan int, 3)

		err := pool.ParallelDo(
			func() error {
				resultsChan <- 1
				return nil
			},
			func() error {
				resultsChan <- 2
				return nil
			},
			func() error {
				resultsChan <- 3
				return nil
			},
		)
		Expect(err).ToNot(HaveOccurred())
		close(resultsChan)

		results := []int{}
		for result := range resultsChan {
			results = append(results, result)
		}
		Expect(results).To(ContainElement(1))
		Expect(results).To(ContainElement(2))
		Expect(results).To(ContainElement(3))
	})

	It("bubbles up any errors", func() {
		pool := work.Pool{
			Count: 2,
		}

		err := pool.ParallelDo(
			func() error {
				return nil
			},
			func() error {
				return bosherr.ComplexError{
					Err:   errors.New("fake-error"),
					Cause: errors.New("fake-cause"),
				}
			},
			func() error {
				return nil
			},
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-error"))
		Expect(err.Error()).To(ContainSubstring("fake-cause"))
	})

	It("stops working after the first error", func() {
		pool := work.Pool{
			Count: 1, // Force serial run
		}

		err := pool.ParallelDo(
			func() error {
				return nil
			},
			func() error {
				return bosherr.ComplexError{
					Err:   errors.New("fake-error"),
					Cause: errors.New("fake-cause"),
				}
			},
			func() error {
				Fail("Expected third test to not run")
				return nil
			},
			func() error {
				Fail("Expected fourth test to not run")
				return nil
			},
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-error"))
		Expect(err.Error()).To(ContainSubstring("fake-cause"))
	})
})
