package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func supportsExternalPowerCheck() error {
	_, err := exec.LookPath("ioreg")
	if err != nil {
		return fmt.Errorf("'%w' External power detection on MacOS requires the ioreg command. This should be installed by default (?)", err)
	}
	return nil
}

func checkIsConnectedToExternalPower() (bool, error) {
	buf := new(bytes.Buffer)

	c := exec.Command("ioreg", "-c", "AppleSmartBattery", "-r")
	c.Stdout = bufio.NewWriter(buf)

	err := c.Run()
	if err != nil {
		return false, err
	}

	return strings.Contains(buf.String(), "\"ExternalConnected\" = Yes"), nil
}
