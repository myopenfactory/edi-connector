// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

var (
	imageURL = "myopenfactory.azurecr.io/images/protobuf"
	services = []string{"api"}
	name     = "myof-client"
)

func init() {
	switch runtime.GOOS {
	case "windows":
		name = name + ".exe"
	}
}

func Build() error {
	fmt.Println("Building...")
	date := time.Now().Format(time.RFC3339)
	commit, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return err
	}
	ldflags := fmt.Sprintf("-X github.com/myopenfactory/client/cmd.Date=%s -X github.com/myopenfactory/client/cmd.Commit=%s", date, strings.TrimSpace(string(commit)))
	return sh.RunV("go", "build", "-o", name, "-ldflags", ldflags, ".")
}

func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll(name)
	for _, service := range services {
		os.Remove(fmt.Sprintf("api/%s.twirp.go", service))
		os.Remove(fmt.Sprintf("api/%s.pb.go", service))
	}
	os.Remove("api/version.txt")
}

func Protogen() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	var protos []string
	for _, service := range services {
		protos = append(protos, fmt.Sprintf("%s.proto", service))
	}
	err = sh.RunV("docker", "run", "--rm", "--platform=linux", "-v", fmt.Sprintf("%s:/data", filepath.Join(dir, "api")), imageURL, "-I", "/data", "--go_out=/data", "--twirp_out=/data", strings.Join(protos, " "))
	if err != nil {
		return err
	}

	files, err := filepath.Glob("api/*.twirp.go")
	if err != nil {
		return err
	}

	for _, file := range files {
		in, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		out := strings.Replace(string(in), `"/twirp`, `"/v1`, -1)
		if err := ioutil.WriteFile(file, []byte(out), 0); err != nil {
			return err
		}
	}
	return nil
}

func Test() error {
	return sh.RunV("mage", "-d", "test", "test")
}
