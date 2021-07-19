package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func checkError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCommand(name string, args ...string) error {
	fmt.Print(name)
	for _, a := range args {
		if strings.ContainsRune(a, ' ') {
			fmt.Print(" ", strconv.Quote(a))
		} else {
			fmt.Print(" ", a)
		}
	}
	fmt.Println()
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	start := time.Now()
	defer func() { fmt.Println("finished in", time.Since(start)) }()
	return cmd.Run()
}

func getVersion() string {
	var tag string
	versionInfoBy, err := os.ReadFile("versioninfo.json")
	if err == nil {
		type version struct {
			Major uint8
			Minor uint8
			Patch uint8
		}
		type fixedFileInfo struct {
			FileVersion version
		}
		type info struct {
			FixedFileInfo fixedFileInfo
		}
		var versionInfo info
		err = json.Unmarshal(versionInfoBy, &versionInfo)
		if err == nil {
			tag = fmt.Sprint(versionInfo.FixedFileInfo.FileVersion.Major, ".",
				versionInfo.FixedFileInfo.FileVersion.Minor, ".",
				versionInfo.FixedFileInfo.FileVersion.Patch)
		}
	}
	if tag == "" {
		tag = "v0.0.0"
	}
	var sha string
	shaByte, err := exec.Command("git", "rev-list", "-1", "--abbrev=7", "--abbrev-commit", "HEAD").Output()
	if err != nil {
		sha = "(not versioned)"
	} else {
		sha = strings.TrimSpace(string(shaByte))
	}
	return fmt.Sprint(tag, "-", sha)
}

func main() {
	start := time.Now()
	defer func() { fmt.Println("completely finished in", time.Since(start)) }()
	checkError(runCommand("go", "get", "-d", "-v"))
	checkError(runCommand("go", "generate", "-v"))
	checkError(runCommand("go", "build", "-v", "-trimpath", "-ldflags",
		fmt.Sprintf(`-w -s -X "main.version=%s" -X "main.buildTime=%s" -H windowsgui`, getVersion(), time.Now().Format(time.RFC822))))
	// checkError(runCommand("go", "test", "-v"))
}
