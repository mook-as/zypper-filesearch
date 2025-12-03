// SPDX-License-Identifier: GPL-2.0-or-later
// SPDX-FileCopyrightText: SUSE LLC

// Package zypper wraps interactions with zypper to find repositories enabled
// on the system.
package zypper

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

type Repository struct {
	Alias   string `xml:"alias,attr"`
	Name    string `xml:"name,attr"`
	Type    string `xml:"type,attr"`
	Enabled bool   `xml:"enabled,attr"`
	URL     string `xml:"url"`
}

var arch = sync.OnceValues(func() (string, error) {
	var buf bytes.Buffer
	cmd := exec.Command("zypper", "system-architecture")
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
})

// List the repositories that are enabled on the system.
func ListRepositories(ctx context.Context, releaseVer string) ([]*Repository, error) {
	var buf bytes.Buffer
	args := []string{"--xmlout", "repos"}
	if releaseVer != "" {
		args = append([]string{"--releasever", releaseVer}, args...)
	}
	cmd := exec.CommandContext(ctx, "zypper", args...)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	var data struct {
		Repos []*Repository `xml:"repo-list>repo"`
	}
	if err := xml.Unmarshal(buf.Bytes(), &data); err != nil {
		return nil, fmt.Errorf("failed to parse repositories: %w", err)
	}

	for _, repo := range data.Repos {
		if repo.Type == "" {
			// Assume rpm-md if no type given
			repo.Type = "rpm-md"
		}
	}

	return data.Repos, nil
}

func Arch() (string, error) {
	return arch()
}
