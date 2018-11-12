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
	return sh.RunV("go", "build", "-o", name, ".")
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

func CI() error {
	for _, goos := range []string{"linux", "windows"} {
		for _, goarch := range []string{"amd64", "386"} {
			ldflags := fmt.Sprintf("\" -X github.com/myopenfactory/client/cmd.version=%s\"", "1.0.0")
			name := fmt.Sprintf("myof-client_%s_%s", goos, goarch)
			if goos == "windows" {
				name += ".exe"
			}
			cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", name)
			cmd.Env = append(os.Environ(),
				"CGO_ENABLED=false",
				fmt.Sprintf("GOOS=%s", goos),
				fmt.Sprintf("GOARCH=%s", goarch),
			)
			if err := cmd.Run(); err != nil {
				return err
			}
		}
	}
	return nil
}
