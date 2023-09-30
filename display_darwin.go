package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func supportsDisplayCheck() error {
	_, err := exec.LookPath("brightness")
	if err != nil {
		return fmt.Errorf("'%w' Display detection on MacOS requires the brightness command. https://github.com/nriley/brightness", err)
	}
	return nil
}

func checkDisplay() (bool, error) {
	buf := new(bytes.Buffer)

	c := exec.Command("brightness", "-l")
	c.Stdout = bufio.NewWriter(buf)

	err := c.Run()
	if err != nil {
		return false, err
	}

	return strings.Contains(buf.String(), "awake"), nil
}
