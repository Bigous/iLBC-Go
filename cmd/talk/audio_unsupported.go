//go:build !windows && !linux && !darwin

package main

import "errors"

func startAudio(*audioBridge) (func(), error) {
	return nil, errors.New("talk supports Windows, Linux, and macOS")
}
