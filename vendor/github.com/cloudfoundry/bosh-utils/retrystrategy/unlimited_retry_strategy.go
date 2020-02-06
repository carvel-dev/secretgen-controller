package retrystrategy

import (
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type unlimitedRetryStrategy struct {
	maxAttempts int
	delay       time.Duration
	retryable   Retryable
	logger      boshlog.Logger
	logTag      string
}

func NewUnlimitedRetryStrategy(
	delay time.Duration,
	retryable Retryable,
	logger boshlog.Logger,
) RetryStrategy {
	return &unlimitedRetryStrategy{
		delay:     delay,
		retryable: retryable,
		logger:    logger,
		logTag:    "unlimitedRetryStrategy",
	}
}

func (s *unlimitedRetryStrategy) Try() error {
	var err error
	var shouldRetry bool
	for i := 0; ; i++ {
		s.logger.Debug(s.logTag, "Making attempt #%d", i)

		shouldRetry, err = s.retryable.Attempt()
		if !shouldRetry {
			return err
		}

		time.Sleep(s.delay)
	}
}
