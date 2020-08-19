package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pkg/errors"
)

type MockClient struct{}

var (
	DoFunc  func(req *http.Request) (*http.Response, error)
	GetFunc func(url string) (*http.Response, error)
)

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return DoFunc(req)
}

func (m *MockClient) Get(url string) (resp *http.Response, err error) {
	return GetFunc(url)
}

func TestMain(m *testing.M) {
	client = &MockClient{}
	os.Exit(m.Run())
}

func TestAuthStartError(t *testing.T) {
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

func TestAuthStartData(t *testing.T) {
	d := DeviceCode{
		DeviceCode:              "0",
		UserCode:                "test",
		VerificationURI:         "test",
		ExpiresIn:               1,
		Interval:                1,
		VerificationURIComplete: "test11",
	}
	DoFunc = func(req *http.Request) (*http.Response, error) {
		header := req.Header.Values("content-type")
		if len(header) != 1 {
			return nil, errors.New("content-type len is wrong")
		}

		if header[0] != "application/x-www-form-urlencoded" {
			return nil, errors.New("content-type is wrong")
		}

		if req.Method != "POST" {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "/oauth/device/code") {
			return nil, errors.New("url is wrong")
		}

		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, body)
		if err != nil {
			return nil, err
		}

		if !strings.Contains(buf.String(), "client_id=") ||
			!strings.Contains(buf.String(), "&audience=https://api.arduino.cc") {
			return nil, errors.New("Payload is wrong")
		}

		data, err := json.Marshal(d)
		if err != nil {
			return nil, err
		}

		return &http.Response{
			Body: ioutil.NopCloser(bytes.NewBufferString(string(data))),
		}, nil
	}

	data, err := StartDeviceAuth("", "0")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, data, d)
}

func TestAuthCheck(t *testing.T) {
	type AuthAccess struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	DoFunc = func(req *http.Request) (*http.Response, error) {
		header := req.Header.Values("content-type")
		if len(header) != 1 {
			return nil, errors.New("content-type len is wrong")
		}

		if header[0] != "application/x-www-form-urlencoded" {
			return nil, errors.New("content-type is wrong")
		}

		if req.Method != "POST" {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "/oauth/token") {
			return nil, errors.New("url is wrong")
		}

		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, body)
		if err != nil {
			return nil, err
		}

		if !strings.Contains(buf.String(), "client_id=") ||
			!strings.Contains(buf.String(), "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=") {
			return nil, errors.New("Payload is wrong")
		}

		data, err := json.Marshal(AuthAccess{
			AccessToken: "asdf",
			ExpiresIn:   999,
			TokenType:   "testType",
		})
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(string(data))),
		}, nil
	}

	token, err := CheckDeviceAuth("", "0", "")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "asdf", token)
}

func TestDefaultConfig(t *testing.T) {
	c := New()
	defaultConfig := &Config{
		CodeURL:     "https://hydra.arduino.cc/oauth2/auth",
		TokenURL:    "https://hydra.arduino.cc/oauth2/token",
		ClientID:    "cli",
		RedirectURI: "http://localhost:5000",
		Scopes:      "profile:core offline",
	}

	assert.Equal(t, defaultConfig, c)
}

func TestRequestAuthError(t *testing.T) {
	config := Config{
		CodeURL: "www.test.com",
	}

	GetFunc = func(url string) (*http.Response, error) {
		return nil, errors.New("test error")
	}

	_, _, err := config.requestAuth(client)
	if err == nil {
		t.Error(err)
	}

	assert.True(t, err != nil)
}

func TestRequestAuth(t *testing.T) {
	config := Config{
		ClientID:    "1",
		CodeURL:     "www.test.com",
		RedirectURI: "http://localhost:5000",
		Scopes:      "profile:core offline",
	}

	coutGetCall := 0
	GetFunc = func(url string) (*http.Response, error) {
		coutGetCall++

		if coutGetCall == 1 {
			if !strings.Contains(url, "www.test.com?client_id=1&redirect_uri=http%3A%2F%2Flocalhost%3A5000&response_type=code&scope=profile%3Acore+offline&state=") {
				return nil, errors.New("Error in url")
			}

			return &http.Response{
				StatusCode: 200,
			}, nil
		}

		if url != "" {
			return nil, errors.New("url should be empty because no Location is provided in Header")
		}

		r, err := http.NewRequest("GET", "www.test.com", bytes.NewBufferString(""))
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: 200,
			Request:    r,
		}, nil
	}

	res, biscuits, err := config.requestAuth(client)
	if err != nil {
		t.Error(err)
	}

	expectedCookies := cookies{}
	expectedCookies["hydra"] = []*http.Cookie{}
	expectedCookies["auth"] = []*http.Cookie{}
	assert.Equal(t, expectedCookies, biscuits)
	assert.Equal(t, "www.test.com", res)
}

func TestAuthenticateError(t *testing.T) {
	config := Config{
		ClientID:    "1",
		CodeURL:     "www.test.com",
		RedirectURI: "http://localhost:5000",
		Scopes:      "profile:core offline",
	}

	DoFunc = func(req *http.Request) (*http.Response, error) {
		header := req.Header.Values("content-type")
		if len(header) != 1 {
			return nil, errors.New("content-type len is wrong")
		}

		if header[0] != "application/x-www-form-urlencoded" {
			return nil, errors.New("content-type is wrong")
		}

		if req.Method != "POST" {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "www.test.io") {
			return nil, errors.New("url is wrong")
		}

		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		buf := new(strings.Builder)
		_, err = io.Copy(buf, body)
		if err != nil {
			return nil, err
		}

		if !strings.Contains(buf.String(), "username=") ||
			!strings.Contains(buf.String(), "password") ||
			!strings.Contains(buf.String(), "csrf") ||
			!strings.Contains(buf.String(), "g-recaptcha-response") {
			return nil, errors.New("Payload is wrong")
		}

		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       ioutil.NopCloser(bytes.NewBufferString(string("testBody"))),
		}, nil
	}

	response, err := config.authenticate(client, cookies{}, "www.test.io", "user", "pw")
	if err == nil {
		t.Error("This test should return an error")
	}

	assert.Equal(t, "", response)
}

func TestAuthenticate(t *testing.T) {
	config := Config{
		ClientID:    "1",
		CodeURL:     "www.test.com",
		RedirectURI: "http://localhost:5000",
		Scopes:      "profile:core offline",
	}

	countDo := 0
	DoFunc = func(req *http.Request) (*http.Response, error) {
		countDo++
		if countDo == 1 {
			header := req.Header.Values("content-type")
			if len(header) != 1 {
				return nil, errors.New("content-type len is wrong")
			}

			if header[0] != "application/x-www-form-urlencoded" {
				return nil, errors.New("content-type is wrong")
			}

			if req.Method != "POST" {
				return nil, errors.New("Method is wrong")
			}

			if !strings.Contains(req.URL.Path, "www.test.io") {
				return nil, errors.New("url is wrong")
			}

			body, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			buf := new(strings.Builder)
			_, err = io.Copy(buf, body)
			if err != nil {
				return nil, err
			}

			if !strings.Contains(buf.String(), "username=") ||
				!strings.Contains(buf.String(), "password") ||
				!strings.Contains(buf.String(), "csrf") ||
				!strings.Contains(buf.String(), "g-recaptcha-response") {
				return nil, errors.New("Payload is wrong")
			}

			return &http.Response{
				StatusCode: 302,
				Status:     "302 OK",
				Body:       ioutil.NopCloser(bytes.NewBufferString(string("testBody"))),
			}, nil
		}

		if req.Method != "GET" {
			return nil, errors.New("Method is wrong")
		}

		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       ioutil.NopCloser(bytes.NewBufferString(string("testBody"))),
		}, nil
	}

	response, err := config.authenticate(client, cookies{}, "www.test.io", "user", "pw")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "", response)
}
