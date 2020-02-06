package retrystrategy

import (
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/jpillora/backoff"
)

type backoffWithJitterRetryStrategy struct {
	maxAttempts int
	min time.Duration
	max time.Duration
	retryable   Retryable
	logger      boshlog.Logger
	logTag      string
}

func NewBackoffWithJitterRetryStrategy(
	maxAttempts int,
	min time.Duration,
	max time.Duration,
	retryable Retryable,
	logger boshlog.Logger,
) RetryStrategy {
	return &backoffWithJitterRetryStrategy{
		maxAttempts: maxAttempts,
		min:         min,
		max:         max,
		retryable:   retryable,
		logger:      logger,
		logTag:      "backoffWithJitterRetryStrategy",
	}
}

func (s *backoffWithJitterRetryStrategy) Try() error {
	var err error
	var shouldRetry bool

	b := &backoff.Backoff{
		Min:    s.min,
		Max:    s.max,
		Factor: 2,
		Jitter: true,
	}

	for i := 0; i < s.maxAttempts; i++ {
		s.logger.Debug(s.logTag, "Making attempt #%d for %T", i, s.retryable)

		shouldRetry, err = s.retryable.Attempt()
		if !shouldRetry {
			return err
		}

		time.Sleep(b.Duration())
	}

	return err
}
