package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mbndr/logo"
	yaml "gopkg.in/yaml.v2"
)

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
)

// Exported var used as mapping on CVE severity name to implied ranking
var SeverityMap = map[string]int{
	"Critical":   1,
	"High":       2,
	"Medium":     3,
	"Low":        4,
	"Negligible": 5,
	"Unknown":    6,
}

// listenForSignal listens for interactions and executes the desired code when it happens
func listenForSignal(fn func(os.Signal)) {
	signalChannel := make(chan os.Signal, 0)

	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGQUIT)
	for {
		execute := <-signalChannel
		fn(execute)
	}
}

// createTmpPath creates a temporary folder with a prefix
func createTmpPath(logger *logo.Logger, tmpPrefix string) string {
	tmpPath, err := os.MkdirTemp("", tmpPrefix)
	if err != nil {
		logger.Fatalf("Could not create temporary folder: %s", err)
	}
	return tmpPath
}

// untar uses a Reader that represents a tar to untar it on the fly to a target folder
func untar(imageReader io.ReadCloser, target string) error {
	tarReader := tar.NewReader(imageReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		if !strings.HasPrefix(path, filepath.Clean(target)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: illegal file path", header.Name)
		}
		info := header.FileInfo()
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err = io.Copy(file, tarReader); err != nil {
			return err
		}
	}
	return nil
}

// parseWhitelistFile reads the whitelist file and parses it
func parseWhitelistFile(logger *logo.Logger, whitelistFile string) vulnerabilitiesWhitelist {
	whitelistTmp := vulnerabilitiesWhitelist{}

	whitelistBytes, err := os.ReadFile(whitelistFile)
	if err != nil {
		logger.Fatalf("Could not parse whitelist file, could not read file %v", err)
	}
	if err = yaml.Unmarshal(whitelistBytes, &whitelistTmp); err != nil {
		logger.Fatalf("Could not parse whitelist file, could not unmarshal %v", err)
	}
	return whitelistTmp
}

// Validate that the given CVE severity threshold is a valid severity
func validateThreshold(logger *logo.Logger, threshold string) {
	for severity := range SeverityMap {
		if threshold == severity {
			return
		}
	}
	logger.Fatalf("Invalid CVE severity threshold %s given", threshold)
}
