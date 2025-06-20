package main

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"strings"
)

func run(dir string, args ...string) ([]byte, error) {
	stdoutBuf := bytes.Buffer{}
	log.Printf("running %s in %s", strings.Join(args, " "), dir)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Printf("stderr: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	return stdoutBuf.Bytes(), nil
}
