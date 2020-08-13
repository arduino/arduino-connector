package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pkg/errors"
)

type MockClient struct{}

var (
	DoFunc func(req *http.Request) (*http.Response, error)
)

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return DoFunc(req)
}

func TestStartAuthError(t *testing.T) {
	client = &MockClient{}

	DoFunc = func(*http.Request) (*http.Response, error) {
		return nil, errors.New(
			"Wanted error from mock web server",
		)
	}

	data, err := StartDeviceAuth("", "0")
	if err == nil {
		t.Error(err)
	}

	assert.Equal(t, data, DeviceCode{})
	assert.NotNil(t, err)
}
