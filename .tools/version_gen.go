//go:build ignore
// +build ignore

// This program generates version.go. It can be invoked by running invoking go:generate
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

func main() {
	bumpType := os.Getenv("BUMP_TYPE")
	versionStem := []byte(os.Getenv("VERSION"))
	var err error
	if bumpType != "" {
		fmt.Println("predicting the next version; mode=", bumpType)
		versionStem, err = exec.Command("sbot", "predict", "version", "--mode", bumpType).Output()
	} else if len(versionStem) > 0 {
		fmt.Println("using provided version:", string(versionStem))
	} else {
		versionStem, err = exec.Command("sbot", "get", "version").Output()
	}
	if err != nil {
		fmt.Printf("can't read version: %s", err)
		fmt.Printf("defaulting to 0.0.1")
		versionStem = []byte("0.0.1")
	}

	if len(os.Args) < 2 {
		fmt.Printf("usage: go run intneral/version_gen.go <appName>\n\nrun from project root")
		os.Exit(1)
	}

	gitSHA, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	//fmt.Printf("sha: >%s< (err: %#v)\n", string(gitSHA), err)
	if err != nil {
		panic(err)
	}

	var hasStaged bool
	if _, err := exec.Command("git", "diff-index", "--quiet", "--cached", "HEAD", "--").Output(); err != nil {
		hasStaged = true
	}

	var hasModified bool
	if _, err := exec.Command("git", "diff-files", "--quiet").Output(); err != nil {
		hasModified = true
	}

	var hasUntracked bool
	if _, err := exec.Command("git", "ls-files", "--exclude-standard", "--others").Output(); err != nil {
		hasUntracked = true
	}

	var isDirty = hasStaged || hasModified || hasUntracked

	gitBranch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	//fmt.Printf("branch: >%s< (err: %#v)\n", string(gitBranch), err)
	if err != nil {
		panic(err)
	}
	gitBranch = bytes.Trim(gitBranch, "\r\n")

	commitsBetweenHeadAndMain, err := exec.Command("git", "rev-list", "origin/main...HEAD").Output()
	if err != nil {
		panic(err)
	}
	if !isDirty && !strings.EqualFold(string(gitBranch), "main") && len(commitsBetweenHeadAndMain) == 0 {
		fmt.Printf("no difference between '%s' and origin/main; replacing '%s' with 'main'\n", string(gitBranch), string(gitBranch))
		gitBranch = []byte("main")
	}
	summary := []byte(os.Getenv("RELEASE_COMMIT_MESSAGE"))
	if len(summary) == 0 {
		summary, err = exec.Command("git", "log", "-1", "--pretty=%s").Output()
		if err != nil {
			panic(err)
		}
	}

	_, err = exec.Command("mkdir", "-p", "./internal/version").Output()
	if err != nil {
		panic(err)
	}

	fmt.Printf("generating/updating: ./internal/version/detail.go\n")
	f, err := os.Create("internal/version/detail.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	coreVersion := string(bytes.Trim(versionStem, "\r\n"))
	branchName := string(bytes.Trim(gitBranch, "\r\n"))
	sha := string(bytes.Trim(gitSHA, "\r\n"))
	semver := generateSemanticVersion(coreVersion, branchName, sha, isDirty)

	packageTemplate.Execute(f, struct {
		AppName         string
		CoreVersion     string
		CommitSummary   string
		BranchName      string
		SHA             string
		IsDirty         bool
		HasStaged       bool
		HasModified     bool
		HasUntracked    bool
		SemanticVersion string
		Timestamp       time.Time
	}{
		AppName:         os.Args[1],
		CoreVersion:     coreVersion,
		CommitSummary:   string(bytes.Trim(summary, "\r\n")),
		BranchName:      branchName,
		IsDirty:         isDirty,
		HasStaged:       hasStaged,
		HasModified:     hasModified,
		HasUntracked:    hasUntracked,
		SemanticVersion: semver,
		SHA:             sha,
		Timestamp:       time.Now(),
	})
}

func generateSemanticVersion(coreVersion string, branchName string, gitSHA string, dirty bool) string {
	v := coreVersion
	usedBranchName := false

	// append preReleaseIdentifier if needed
	if strings.EqualFold(branchName, "main") {
		usedBranchName = true
	} else {
		preReleaseIdentifier := "alpha"
		if strings.HasPrefix(branchName, "release-") {
			preReleaseIdentifier = branchNameToBuildMetadataSegment(branchName[len("release-"):])
			usedBranchName = true
		} else if strings.HasPrefix(branchName, "rc") {
			preReleaseIdentifier = branchName
			usedBranchName = true
		}
		v = v + "-" + preReleaseIdentifier
	}

	// append build metadata: SHA
	v += "+" + gitSHA

	// add branch name (unless used as a pre-release identifier)
	if !usedBranchName {
		v += "." + branchNameToBuildMetadataSegment(branchName)
	}

	// add dirty flag if needed
	if dirty {
		v += ".dirty"
	}

	return v
}

func branchNameToBuildMetadataSegment(name string) string {
	return strings.Replace(name, "_", "-", -1)
}

var packageTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at {{ .Timestamp }}
package version

import (
	"fmt"
	"os/user"
	"runtime"
	"sort"
	"strings"
)

// Detail provides an easy global way to
var Detail = NewVersionDetail()

// NewVersionDetail builds a new version DetailStruct
func NewVersionDetail() DetailStruct {
	s := DetailStruct{
		AppName:              "{{ .AppName }}",
		BuildDate:            "{{ .Timestamp }}",
		CoreVersion:          "{{ .CoreVersion }}",
		GitBranch:            "{{ .BranchName }}",
		GitCommit:            "{{ .SHA }}",
		GitCommitSummary:     "{{ .CommitSummary }}",
		GitDirty:             {{ .IsDirty }},
		GitDirtyHasModified:  {{ .HasModified }},
		GitDirtyHasStaged:    {{ .HasStaged }},
		GitDirtyHasUntracked: {{ .HasUntracked }},
		Version:              "{{ .SemanticVersion }}",
	}
	s.UserAgentString = s.ToUserAgentString()
	if s.GitDirty {
		s.GitWorkingState = "dirty"
	}
	return s
}

// DetailStruct provides an easy way to grab all the govvv version details together
type DetailStruct struct {
	AppName              string ` + "`json:\"app_name\"`" + `
	BuildDate            string ` + "`json:\"build_date\"`" + `
	CoreVersion          string ` + "`json:\"core_version\"`" + `
	GitBranch            string ` + "`json:\"branch\"`" + `
	GitCommit            string ` + "`json:\"commit\"`" + `
	GitCommitSummary     string ` + "`json:\"commit_summary\"`" + `
	GitDirty             bool ` + "`json:\"dirty\"`" + `
	GitDirtyHasModified  bool ` + "`json:\"dirty_modified\"`" + `
	GitDirtyHasStaged    bool ` + "`json:\"dirty_staged\"`" + `
	GitDirtyHasUntracked bool ` + "`json:\"dirty_untracked\"`" + `
	GitWorkingState      string ` + "`json:\"working_state\"`" + `
	GitSummary           string ` + "`json:\"summary\"`" + `
	UserAgentString      string ` + "`json:\"user_agent\"`" + `
	Version              string ` + "`json:\"version\"`" + `
}

// String implements Stringer
func (d *DetailStruct) String() string {
	if d == nil {
		return "n/a"
	}
	return fmt.Sprintf("%s %s", d.AppName, d.Version)
}

// ToUserAgentString formats a DetailStruct as a User-Agent string
func (s DetailStruct) ToUserAgentString() string {
	productName := s.AppName
	productVersion := s.Version

	productDetails := map[string]string{ }

	user, err := user.Current()
	if err == nil {
		username := user.Username
		if username == "" {
			username = "unknown"
		}
	}

	detailParts := []string{}
	for k, v := range productDetails {
		detailParts = append(detailParts, fmt.Sprintf("%s: %s", k, v))
	}
	sort.Slice(detailParts, func(i, j int) bool {
		return detailParts[i] < detailParts[j]
	})
	productDetail := strings.Join(detailParts, ", ")

	return fmt.Sprintf("%s/%s (%s) %s (%s)", productName, productVersion, productDetail, runtime.GOOS, runtime.GOARCH)
}
`))
