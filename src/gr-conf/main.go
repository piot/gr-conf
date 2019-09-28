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
	"net/url"
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
	Organization string `required:"" help:"Organization name on github"`
	Directory    string `required:"" help:"work directory for source files"`
}

func (c *FetchCmd) Run(log *clog.Log) error {
	return run(c.Organization, c.Directory, log)
}

type Options struct {
	Fetch   FetchCmd    `cmd:""`
	Options cli.Options `embed:""`
}

func repoIsGo(repo *github.Repository) bool {
	return *repo.Language == "Go"
}

func getFilePath(prefix string, goSourceDirectory string, repo *github.Repository) (string, error) {
	if repoIsGo(repo) {
		repoURL, parseErr := url.Parse(*repo.CloneURL)
		if parseErr != nil {
			return "", parseErr
		}
		suffix := repoURL.Host + repoURL.Path[:len(repoURL.Path)-4]
		prefix := path.Join(goSourceDirectory, suffix)
		return prefix, nil
	}
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
	log.Debug("cloning repo", clog.String("cloneUrl", repoURL), clog.String("targetPath", complete))
	_, executeErr := execute(log, "git", "clone", repoURL, complete)
	return executeErr
}

func run(organizationName string, targetDirectory string, log *clog.Log) error {
	pathToGo := os.Getenv("GOPATH")
	if pathToGo == "" {
		return fmt.Errorf("GOPATH must be set")
	}
	goSourceDirectory := path.Join(pathToGo, "src/")
	const isUser = true
	repos, reposErr := grconf.Fetch(organizationName, isUser, log)
	if reposErr != nil {
		return reposErr
	}
	for _, repo := range repos {
		if *repo.Archived {
			continue
		}
		if repo.Language == nil {
			log.Warn("no language set", clog.String("name", *repo.Name))
			continue
		}
		log.Trace("found repo", clog.String("repo", *repo.Name), clog.String("language", *repo.Language))
		complete, completeErr := getFilePath(targetDirectory, goSourceDirectory, repo)
		if completeErr != nil {
			return completeErr
		}
		log.Trace("complete path", clog.String("path", complete))
		if _, err := os.Stat(complete); !os.IsNotExist(err) {
			log.Info("directory already exists, skipping", clog.String("directory", complete))
		} else {
			gitClone(*repo.CloneURL, complete, log)
		}
	}

	return nil
}

func main() {
	cli.Run(&Options{}, cli.RunOptions{ApplicationType: cli.Utility, Version: Version})
}
