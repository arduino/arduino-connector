package auth

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/textproto"
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
	DoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("Wanted error from mock web server")
	}

	data, err := StartDeviceAuth("", "0")

	assert.Error(t, err)
	assert.Equal(t, data, DeviceCode{})
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
		header := req.Header[textproto.CanonicalMIMEHeaderKey("content-type")]
		if len(header) != 1 {
			return nil, errors.New("content-type len is wrong")
		}

		if header[0] != "application/x-www-form-urlencoded" {
			return nil, errors.New("content-type is wrong")
		}

		if req.Method != http.MethodPost {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "/oauth/device/code") {
			return nil, errors.New("url is wrong")
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		bodyStr := string(body)
		if !strings.Contains(bodyStr, "client_id=") ||
			!strings.Contains(bodyStr, "&audience=https://api.arduino.cc") {
			return nil, errors.New("Payload is wrong")
		}

		var data bytes.Buffer
		if err := json.NewEncoder(&data).Encode(d); err != nil {
			return nil, err
		}

		return &http.Response{
			Body: ioutil.NopCloser(&data),
		}, nil
	}

	data, err := StartDeviceAuth("", "0")

	assert.NoError(t, err)
	assert.Equal(t, data, d)
}

func TestAuthCheck(t *testing.T) {
	type AuthAccess struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	DoFunc = func(req *http.Request) (*http.Response, error) {
		header := req.Header[textproto.CanonicalMIMEHeaderKey("content-type")]
		if len(header) != 1 {
			return nil, errors.New("content-type len is wrong")
		}

		if header[0] != "application/x-www-form-urlencoded" {
			return nil, errors.New("content-type is wrong")
		}

		if req.Method != http.MethodPost {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "/oauth/token") {
			return nil, errors.New("url is wrong")
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		bodyStr := string(body)
		if !strings.Contains(bodyStr, "client_id=") ||
			!strings.Contains(bodyStr, "grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code=") {
			return nil, errors.New("Payload is wrong")
		}

		var data bytes.Buffer
		if err := json.NewEncoder(&data).Encode(AuthAccess{
			AccessToken: "asdf",
			ExpiresIn:   999,
			TokenType:   "testType",
		}); err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(&data),
		}, nil
	}

	token, err := CheckDeviceAuth("", "0", "")

	assert.NoError(t, err)
	assert.Equal(t, "asdf", token)
}

func TestDefaultConfig(t *testing.T) {
	assert.Equal(t, New(), &Config{
		CodeURL:     "https://hydra.arduino.cc/oauth2/auth",
		TokenURL:    "https://hydra.arduino.cc/oauth2/token",
		ClientID:    "cli",
		RedirectURI: "http://localhost:5000",
		Scopes:      "profile:core offline",
	})
}

func TestRequestAuthError(t *testing.T) {
	config := Config{
		CodeURL: "www.test.com",
	}

	GetFunc = func(url string) (*http.Response, error) {
		return nil, errors.New("Wanted error from mock web server")
	}

	_, _, err := config.requestAuth(client)
	assert.Error(t, err)
}

func TestRequestAuth(t *testing.T) {
	config := Config{
		ClientID:    "1",
		CodeURL:     "www.test.com",
		RedirectURI: "http://localhost:5000",
		Scopes:      "profile:core offline",
	}

	countGetCall := 0
	GetFunc = func(url string) (*http.Response, error) {
		countGetCall++

		if countGetCall == 1 {
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

		r, err := http.NewRequest("GET", "www.test.com", nil)
		if err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: 200,
			Request:    r,
		}, nil
	}

	res, biscuits, err := config.requestAuth(client)

	assert.NoError(t, err)
	assert.Equal(t, biscuits, cookies{
		"hydra": []*http.Cookie{},
		"auth":  []*http.Cookie{},
	})
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
		header := req.Header[textproto.CanonicalMIMEHeaderKey("content-type")]
		if len(header) != 1 {
			return nil, errors.New("content-type len is wrong")
		}

		if header[0] != "application/x-www-form-urlencoded" {
			return nil, errors.New("content-type is wrong")
		}

		if req.Method != http.MethodPost {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "www.test.io") {
			return nil, errors.New("url is wrong")
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		bodyStr := string(body)
		if !strings.Contains(bodyStr, "username=") ||
			!strings.Contains(bodyStr, "password") ||
			!strings.Contains(bodyStr, "csrf") ||
			!strings.Contains(bodyStr, "g-recaptcha-response") {
			return nil, errors.New("Payload is wrong")
		}

		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       ioutil.NopCloser(strings.NewReader("testBody")),
		}, nil
	}

	response, err := config.authenticate(client, cookies{}, "www.test.io", "user", "pw")

	assert.Error(t, err)
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
			header := req.Header[textproto.CanonicalMIMEHeaderKey("content-type")]
			if len(header) != 1 {
				return nil, errors.New("content-type len is wrong")
			}

			if header[0] != "application/x-www-form-urlencoded" {
				return nil, errors.New("content-type is wrong")
			}

			if req.Method != http.MethodPost {
				return nil, errors.New("Method is wrong")
			}

			if !strings.Contains(req.URL.Path, "www.test.io") {
				return nil, errors.New("url is wrong")
			}

			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}

			bodyStr := string(body)
			if !strings.Contains(bodyStr, "username=") ||
				!strings.Contains(bodyStr, "password") ||
				!strings.Contains(bodyStr, "csrf") ||
				!strings.Contains(bodyStr, "g-recaptcha-response") {
				return nil, errors.New("Payload is wrong")
			}

			resp := &http.Response{
				StatusCode: 302,
				Status:     "302 OK",
				Header:     http.Header{},
				Body:       ioutil.NopCloser(strings.NewReader("testBody")),
			}
			resp.Header.Set("Location", "www.redirect.io")

			return resp, nil
		}

		if req.Method != http.MethodGet {
			return nil, errors.New("Method is wrong")
		}

		if !strings.Contains(req.URL.Path, "www.redirect.io") {
			return nil, errors.New("redirect url is wrong")
		}

		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       ioutil.NopCloser(strings.NewReader("testBody")),
		}, nil
	}

	response, err := config.authenticate(client, cookies{}, "www.test.io", "user", "pw")

	assert.NoError(t, err)
	assert.Equal(t, "", response)
}

func TestRequestTokenError(t *testing.T) {
	c := Config{}

	DoFunc = func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("Wanted error from mock web server")
	}

	token, err := c.requestToken(client, "9")

	assert.Error(t, err)
	assert.True(t, token == nil)
}

func TestRequestToken(t *testing.T) {
	c := Config{}

	expectedToken := Token{
		Access:  "1234",
		Refresh: "",
		TTL:     99,
		Scopes:  "",
		Type:    "Bearer",
	}

	DoFunc = func(req *http.Request) (*http.Response, error) {
		var data bytes.Buffer
		if err := json.NewEncoder(&data).Encode(expectedToken); err != nil {
			return nil, err
		}

		return &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Body:       ioutil.NopCloser(&data),
		}, nil
	}

	token, err := c.requestToken(client, "9")

	assert.NoError(t, err)
	assert.Equal(t, expectedToken, *token)
}
