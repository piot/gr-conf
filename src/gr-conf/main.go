/*

MIT License

Copyright (c) 2017 Peter Bjorklund

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

*/

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/google/go-github/github"

	"github.com/piot/log-go/src/clog"

	"github.com/piot/cli-go/src/cli"
	grconf "github.com/piot/gr-conf/src/lib"
)

var Version string

type FetchCmd struct {
	Organization string `required:"" help:"Organization or username on github"`
	Directory    string `default:"" help:"work directory for source files"`
}

func (c *FetchCmd) Run(log *clog.Log) error {
	directory := c.Directory
	if directory == "" {
		var directoryErr error
		directory, directoryErr = os.Getwd()
		if directoryErr != nil {
			return directoryErr
		}
	}
	return run(c.Organization, directory, log)
}

type Options struct {
	Fetch   FetchCmd    `cmd:""`
	Options cli.Options `embed:""`
}

func getFilePath(prefix string, repo *github.Repository) (string, error) {
	complete := path.Join(prefix, *repo.Name)
	return complete, nil
}

func execute(log *clog.Log, tool string, args ...string) ([]byte, error) {
	cmd := exec.Command(tool, args...)
	log.Trace("executing", clog.String("tool", tool))
	runErr := cmd.Run()
	if runErr != nil {
		return nil, runErr
	}
	output, outputErr := cmd.CombinedOutput()
	if outputErr != nil {
		return nil, outputErr
	}
	fmt.Printf("%v", string(output))
	return output, nil
}

func gitClone(repoURL string, complete string, log *clog.Log) error {
	log.Info("cloning repo", clog.String("cloneUrl", repoURL), clog.String("targetPath", complete))
	_, executeErr := execute(log, "git", "clone", repoURL, complete)
	return executeErr
}

func run(organizationName string, targetDirectory string, log *clog.Log) error {
	const isUser = true
	repos, reposErr := grconf.Fetch(organizationName, isUser, log)
	if reposErr != nil {
		return reposErr
	}
	for _, repo := range repos {
		if *repo.Archived {
			continue
		}
		log.Trace("found repo", clog.String("repo", *repo.Name))
		complete, completeErr := getFilePath(targetDirectory, repo)
		if completeErr != nil {
			return completeErr
		}
		log.Trace("complete path", clog.String("path", complete))
		if _, err := os.Stat(complete); !os.IsNotExist(err) {
			log.Debug("directory already exists, skipping", clog.String("directory", complete))
		} else {
			gitClone(*repo.CloneURL, complete, log)
		}
	}

	return nil
}

func main() {
	cli.Run(&Options{}, cli.RunOptions{ApplicationType: cli.Utility, Version: Version})
}
