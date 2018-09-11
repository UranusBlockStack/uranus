package params

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

const (
	//Identifier  identifier to advertise over the network
	Identifier = "uranus"
)

const (
	// VersionMajor is Major version component of the current release
	VersionMajor = 0
	// VersionMinor is Minor version component of the current release
	VersionMinor = 1
	// VersionPatch is Patch version component of the current release
	VersionPatch = 0
)

// VersionFunc holds the textual version string.
var VersionFunc = func() string { return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch) }()

// GitCommit  Git SHA1 commit hash of the release
var GitCommit = func() string {
	head := readGit("HEAD")
	if splits := strings.Split(head, " "); len(splits) == 2 {
		head = splits[1]
		return readGit(head)
	}
	return ""
}

// readGit returns content of file in .git directory.
func readGit(file string) string {
	content, err := ioutil.ReadFile(path.Join(".git", file))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}
