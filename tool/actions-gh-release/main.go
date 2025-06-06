// Copyright 2024 The PipeCD Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/go-github/v39/github"
)

const (
	gitExecPath        = "git"
	defaultReleaseFile = "RELEASE"
)

func main() {
	os.Exit(_main())
}

func _main() int {
	log.Println("Start running actions-gh-release")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	args, err := parseArgs(os.Args)
	if err != nil {
		log.Printf("Failed to parse arguments: %v\n", err)
		return 1
	}
	log.Println("Successfully parsed arguments")

	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		log.Printf("GITHUB_WORKSPACE was not defined")
		return 1
	}

	if err := addSafeDirectory(ctx, gitExecPath, workspace); err != nil {
		log.Printf("Failed to add %s as a safe directory: %v\n", workspace, err)
		return 1
	}

	cfg := &githubClientConfig{Token: args.Token}
	ghClient := newGitHubClient(ctx, cfg)

	event, err := ghClient.parseGitHubEvent(ctx)
	if err != nil {
		log.Printf("Failed to parse GitHub event: %v\n", err)
		return 1
	}
	log.Printf("Successfully parsed GitHub event %s\n\tbase-commit %s\n\thead-commit %s\n",
		event.Name,
		event.BaseCommit,
		event.HeadCommit,
	)

	// Find all changed RELEASE files.
	changedFiles, err := changedFiles(ctx, gitExecPath, workspace, event.BaseCommit, event.HeadCommit)
	if err != nil {
		log.Printf("Failed to list changed files: %v\n", err)
		return 1
	}

	changedReleaseFiles := make([]string, 0, len(changedFiles))
	matcher, err := NewPatternMatcher([]string{args.ReleaseFile})
	if err != nil {
		log.Printf("Failed to create pattern matcher for release file: %v\n", err)
		return 1
	}
	for _, f := range changedFiles {
		if matcher.Matches(f) {
			changedReleaseFiles = append(changedReleaseFiles, f)
		}
	}

	if len(changedReleaseFiles) == 0 {
		log.Println("Nothing to do since there were no modified release files")
		return 0
	}

	proposals := make([]ReleaseProposal, 0, len(changedReleaseFiles))
	for _, f := range changedReleaseFiles {
		p, err := buildReleaseProposal(ctx, ghClient, f, gitExecPath, workspace, event)
		if err != nil {
			log.Printf("Failed to build release for %s: %v\n", f, err)
			return 1
		}
		proposals = append(proposals, *p)
	}

	// Filter all existing releases.
	var (
		newProposals      = make([]ReleaseProposal, 0, len(proposals))
		existingProposals = make([]ReleaseProposal, 0)
	)
	for _, p := range proposals {
		if p.Tag != "" {
			exist, err := ghClient.existRelease(ctx, event.Owner, event.Repo, p.Tag)
			if err != nil {
				log.Printf("Failed to check the existence of release for %s: %v\n", p.Tag, err)
				return 1
			}
			if exist {
				existingProposals = append(existingProposals, p)
				continue
			}
		}
		newProposals = append(newProposals, p)
	}

	notify := func() (*github.IssueComment, error) {
		body := makeCommentBody(newProposals, existingProposals)
		return ghClient.sendComment(ctx, event.Owner, event.Repo, event.PRNumber, body)
	}

	if len(existingProposals) != 0 {
		if len(newProposals) == 0 {
			log.Printf("All of %d detected releases were already created before so no new release will be created\n", len(proposals))
			notify()
			return 0
		}
		log.Printf("%d releases from %d detected ones were already created before so only %d releases will be created\n", len(existingProposals), len(proposals), len(newProposals))
	}

	releasesJSON, err := json.Marshal(newProposals)
	if err != nil {
		log.Printf("Failed to marshal releases: %v\n", err)
		return 1
	}
	os.Setenv("GITHUB_OUTPUT", fmt.Sprintf("releases=%s", string(releasesJSON)))
	if args.OutputReleasesFilePath != "" {
		if err := os.WriteFile(args.OutputReleasesFilePath, releasesJSON, 0644); err != nil {
			log.Printf("Failed to write releases JSON to %s: %v\n", args.OutputReleasesFilePath, err)
			return 1
		}
		log.Printf("Successfully wrote releases JSON to %s\n", args.OutputReleasesFilePath)
	}

	// Create GitHub releases if the event was push.
	if event.Name == eventPush {
		log.Printf("Will create %d GitHub releases\n", len(newProposals))
		for _, p := range newProposals {
			r, err := ghClient.createRelease(ctx, event.Owner, event.Repo, p)
			if err != nil {
				log.Printf("Failed to create release %s: %v\n", p.Tag, err)
				return 1
			}
			log.Printf("Successfully created a new GitHub release %s\n%s\n", r.GetTagName(), r.GetHTMLURL())
		}

		log.Printf("Successfully created all %d GitHub releases\n", len(newProposals))
		return 0
	}

	// Otherwise, just leave a comment to show the changelogs.
	comment, err := notify()
	if err != nil {
		log.Printf("Failed to send comment: %v\n", err)
		return 1
	}

	log.Printf("Successfully commented actions-gh-release result on pull request\n%s\n", *comment.HTMLURL)
	return 0
}

type arguments struct {
	ReleaseFile            string
	Token                  string
	OutputReleasesFilePath string
}

func parseArgs(args []string) (arguments, error) {
	var out arguments

	for _, arg := range args {
		ps := strings.SplitN(arg, "=", 2)
		if len(ps) != 2 {
			continue
		}
		switch ps[0] {
		case "release-file":
			out.ReleaseFile = ps[1]
		case "token":
			out.Token = ps[1]
		case "output-releases-file-path":
			out.OutputReleasesFilePath = ps[1]
		}
	}

	if out.ReleaseFile == "" {
		out.ReleaseFile = defaultReleaseFile
	}
	if out.Token == "" {
		return out, fmt.Errorf("token argument must be set")
	}
	return out, nil
}
