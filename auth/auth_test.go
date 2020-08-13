package auth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func Test1(t *testing.T) {
	client := &MockClient{}

	client.DoFunc = func(*http.Request) (*http.Response, error) {
		return nil, errors.New(
			"Wanted error from mocke web server",
		)
	}

	url := "fake"
	payload := strings.NewReader("test")
	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		t.Error(err)
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	fmt.Println(res)
	fmt.Println(err)
}
