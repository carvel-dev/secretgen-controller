package httpclient

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"io"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshuuid "github.com/cloudfoundry/bosh-utils/uuid"
)

type RequestRetryable struct {
	request   *http.Request
	requestID string
	delegate  Client
	attempt   int

	originalBody io.ReadCloser // buffer request body to memory for retries
	response     *http.Response

	uuidGenerator         boshuuid.Generator
	logger                boshlog.Logger
	logTag                string
	isResponseAttemptable func(*http.Response, error) (bool, error)
}

func NewRequestRetryable(
	request *http.Request,
	delegate Client,
	logger boshlog.Logger,
	isResponseAttemptable func(*http.Response, error) (bool, error),
) *RequestRetryable {
	if isResponseAttemptable == nil {
		isResponseAttemptable = defaultIsAttemptable
	}

	return &RequestRetryable{
		request:               request,
		delegate:              delegate,
		attempt:               0,
		uuidGenerator:         boshuuid.NewGenerator(),
		logger:                logger,
		logTag:                "clientRetryable",
		isResponseAttemptable: isResponseAttemptable,
	}
}

func (r *RequestRetryable) Attempt() (bool, error) {
	var err error

	if r.requestID == "" {
		r.requestID, err = r.uuidGenerator.Generate()
		if err != nil {
			return false, bosherr.WrapError(err, "Generating request uuid")
		}
	}

	if r.attempt == 0 {
		r.originalBody, err = MakeReplayable(r.request)
		if err != nil {
			return false, bosherr.WrapError(err, "Ensuring request can be retried")
		}
	} else if r.attempt > 0 && r.request.GetBody != nil {
		r.request.Body, err = r.request.GetBody()
		if err != nil {
			if r.originalBody != nil {
				r.originalBody.Close()
			}

			return false, bosherr.WrapError(err, "Updating request body for retry")
		}
	}

	// close previous attempt's response body to prevent HTTP client resource leaks
	if r.response != nil {
		// net/http response body early closing does not block until the body is
		// properly cleaned up, which would lead to a 'request canceled' error.
		// Yielding the CPU should allow the scheduler to run the cleanup tasks
		// before continuing. But we found that that behavior is not deterministic,
		// we instead avoid the problem altogether by reading the entire body and
		// forcing an EOF.
		// This should not be necessary when the following CL gets accepted:
		// https://go-review.googlesource.com/c/go/+/62891
		io.Copy(ioutil.Discard, r.response.Body)

		r.response.Body.Close()
	}

	r.attempt++

	r.logger.Debug(r.logTag, "[requestID=%s] Requesting (attempt=%d): %s", r.requestID, r.attempt, formatRequest(r.request))
	r.response, err = r.delegate.Do(r.request)

	attemptable, err := r.isResponseAttemptable(r.response, err)
	if !attemptable && r.originalBody != nil {
		r.originalBody.Close()
	}

	return attemptable, err
}

func (r *RequestRetryable) Response() *http.Response {
	return r.response
}

func defaultIsAttemptable(resp *http.Response, err error) (bool, error) {
	if err != nil {
		return true, err
	}

	return !wasSuccessful(resp), nil
}

func wasSuccessful(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < http.StatusMultipleChoices
}

func formatRequest(req *http.Request) string {
	if req == nil {
		return "Request(nil)"
	}

	return fmt.Sprintf("Request{ Method: '%s', URL: '%s' }", req.Method, req.URL)
}

func formatResponse(resp *http.Response) string {
	if resp == nil {
		return "Response(nil)"
	}

	return fmt.Sprintf("Response{ StatusCode: %d, Status: '%s' }", resp.StatusCode, resp.Status)
}

func MakeReplayable(r *http.Request) (io.ReadCloser, error) {
	var err error

	if r.Body == nil {
		return nil, nil
	} else if r.GetBody != nil {
		return nil, nil
	}

	var originalBody = r.Body

	if seekableBody, ok := r.Body.(io.ReadSeeker); ok {
		r.GetBody = func() (io.ReadCloser, error) {
			_, err := seekableBody.Seek(0, 0)
			if err != nil {
				return nil, bosherr.WrapError(err, "Seeking to beginning of seekable request body")
			}

			return ioutil.NopCloser(seekableBody), nil
		}
	} else {
		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return originalBody, bosherr.WrapError(err, "Buffering request body")
		}

		r.GetBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	r.Body, err = r.GetBody()
	if err != nil {
		return originalBody, bosherr.WrapError(err, "Buffering request body")
	}

	return originalBody, nil
}
