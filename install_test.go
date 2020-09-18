package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckCreateConfig(t *testing.T) {
	err := createConfigFolder()
	assert.True(t, err == nil)
	defer func() {
		os.RemoveAll("/etc/arduino-connector")
	}()
	_, err = os.Stat("/etc/arduino-connector")
	assert.True(t, err == nil)
}
