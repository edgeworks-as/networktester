package testers

import (
	"context"
	"edgeworks.no/networktester/api/v1"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

func DoTCPTest(t *v1.Networktest) TestResult {
	timeout, _ := time.ParseDuration(fmt.Sprintf("%ds", t.Spec.Timeout))

	ip := t.Spec.TCP.Address
	port := t.Spec.TCP.Port

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {

		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return TestResult{
				Success: false,
				Message: fmt.Errorf("timeout: %v", err).Error(),
			}
		}

		return TestResult{
			Success: false,
			Message: err.Error(),
		}
	}

	defer conn.Close()

	if t.Spec.TCP.Data == "" {
		return TestResult{
			Success: true,
			Message: conn.RemoteAddr().String(),
		}
	}

	num, err := conn.Write([]byte(t.Spec.TCP.Data))
	if err != nil {
		return TestResult{
			Success: false,
			Message: fmt.Errorf("Failed to write data: %v", err).Error(),
		}
	}

	dataLen := len([]byte(t.Spec.TCP.Data))
	if num != dataLen {
		return TestResult{
			Success: false,
			Message: fmt.Errorf("failed to write data: %d != %d", num, dataLen).Error(),
		}
	}

	return TestResult{
		Success: true,
		Message: conn.RemoteAddr().String(),
	}
}

func DoHttpTest(t *v1.Networktest) TestResult {
	timeout, _ := time.ParseDuration(fmt.Sprintf("%ds", t.Spec.Timeout))
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, t.Spec.Http.URL, nil)
	if err != nil {
		return TestResult{
			Success: false,
			Message: err.Error(),
		}
	}

	tr := &http.Transport{}
	if t.Spec.Http.TlsSkipVerify {
		tr.TLSClientConfig.InsecureSkipVerify = true
	}
	c := http.Client{Transport: tr}

	res, err := c.Do(r)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return TestResult{
				Success: false,
				Message: fmt.Errorf("timeout: %v", err).Error(),
			}
		}

		return TestResult{
			Success: false,
			Message: err.Error(),
		}
	}

	if matchesCode(res.StatusCode, t.Spec.Http.FailOnCodes) {
		return TestResult{
			Success: false,
			Message: fmt.Sprintf("http result: %s matches failOnCodes", res.Status),
		}
	}

	return TestResult{
		Success: true,
		Message: fmt.Sprintf("http result: %s", res.Status),
	}
}

func matchesCode(actualCode int, matchesCodes []int) bool {
	for _, v := range matchesCodes {
		if v == actualCode {
			return true
		}
	}
	return false
}

type TestResult struct {
	Success bool
	Message string
}

const (
	Success = "Success"
	Failed  = "Failed"
)

func (t TestResult) String() *string {
	var res string
	switch t.Success {
	case true:
		res = Success
	default:
		res = Failed
	}
	return &res
}
