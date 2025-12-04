// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Package config handles configuration and loading it from disk.
package config

import (
	"context"
	"flag"
	"path/filepath"
	"slices"

	"github.com/adrg/xdg"
	"gopkg.in/ini.v1"
)

type OutputFormat string

const (
	OutputFormatHuman = OutputFormat("human")
	OutputFormatJSON  = OutputFormat("json")
	OutputFormatXML   = OutputFormat("xml")

	configPath = "zypper-filesearch.conf"
)

type Config struct {
	Verbose    bool
	ReleaseVer string
	Format     OutputFormat
	Enabled    bool
}

var configFromFlags struct {
	verbose    bool
	releaseVer string
	json       bool
	xml        bool
	enabled    bool
}

func AddFlags() {
	flag.BoolVar(&configFromFlags.verbose, "verbose", false, "Enable debug logging")
	flag.StringVar(&configFromFlags.releaseVer, "releasever", "", "Set the value of `zypper --releasever`")
	flag.BoolVar(&configFromFlags.json, "json", false, "Enable JSON output")
	flag.BoolVar(&configFromFlags.xml, "xml", false, "Enable XML output")
	flag.BoolVar(&configFromFlags.enabled, "enabled", true, "Use only enabled repositories")
}

// Read the configuration from disk
func Read(ctx context.Context) (*Config, error) {
	var filePaths []any

	// ini.LoadOptions takes the later paths as more important, but the XDG paths
	// are reversed and the first path is more important; therefore, we need to
	// iterate over some of these backwards.
	for _, dir := range slices.Backward(xdg.DataDirs) {
		filePaths = append(filePaths, filepath.Join(dir, "etc", configPath))
	}
	for _, dir := range slices.Backward(xdg.ConfigDirs) {
		filePaths = append(filePaths, filepath.Join(dir, configPath))
	}
	for _, dir := range []string{"/etc", xdg.ConfigHome} {
		filePaths = append(filePaths, filepath.Join(dir, configPath))
	}
	opts := ini.LoadOptions{Loose: true, Insensitive: true}
	iniFile, err := ini.LoadSources(opts, filePaths[0], filePaths[1:]...)
	if err != nil {
		return nil, err
	}

	section := iniFile.Section("filesearch")
	result := Config{
		Verbose:    section.Key("verbose").MustBool(false),
		ReleaseVer: section.Key("releaseVer").MustString(""),
		Format:     OutputFormat(section.Key("format").MustString("")),
		Enabled:    section.Key("enabled").MustBool(true),
	}
	switch result.Format {
	case OutputFormatJSON, OutputFormatXML:
		// Valid values
	default:
		// Invalid value
		result.Format = OutputFormatHuman
	}

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "verbose":
			result.Verbose = configFromFlags.verbose
		case "releasever":
			result.ReleaseVer = configFromFlags.releaseVer
		case "json":
			if configFromFlags.json {
				result.Format = OutputFormatJSON
			} else {
				result.Format = OutputFormatHuman
			}
		case "xml":
			if configFromFlags.xml {
				result.Format = OutputFormatXML
			} else {
				result.Format = OutputFormatHuman
			}
		case "enabled":
			result.Enabled = configFromFlags.enabled
		}
	})

	return &result, nil
}
