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
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"

	"github.com/google/go-github/github"

	"github.com/piot/log-go/src/clog"

	grconf "github.com/piot/gr-conf/src/lib"
)

func options() (string, string) {
	var organizationName string
	flag.StringVar(&organizationName, "organization", "", "organization name")
	var directory string
	flag.StringVar(&directory, "directory", "", "directory")
	flag.Parse()
	return organizationName, directory
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
	log.Trace("cloning repo", clog.String("cloneUrl", repoURL), clog.String("targetPath", complete))
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
		gitClone(*repo.CloneURL, complete, log)
		log.Trace("complete path", clog.String("path", complete))
	}

	return nil
}

func main() {
	log := clog.DefaultLog()
	organizationName, targetDirectory := options()
	if targetDirectory == "" {
		err := fmt.Errorf("directory must be specified")
		log.Err(err)
		return
	}
	err := run(organizationName, targetDirectory, log)
	if err != nil {
		log.Err(err)
	}
	log.Info("done")
}
