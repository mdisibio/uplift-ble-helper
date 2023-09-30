package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var awake = false

func supportsDisplayCheck() error {
	_, err := exec.LookPath("brightness")
	return fmt.Errorf("'%w' Display detection on MacOS requires the brightness command. https://github.com/nriley/brightness", err)
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

func monitorDisplay() {
	var err error

	fmt.Println("Enabling auto display detection")

	update := func() {
		awake, err = checkDisplay()
		if err != nil {
			fmt.Println("Error detecting display status:", err)
		}

		if awake {
			metricsDisplayOn.Set(1)
		} else {
			metricsDisplayOn.Set(0)
		}
	}

	// Prime status immediately
	update()

	// Then monitor in the background
	go func() {
		t := time.NewTimer(time.Minute)
		for range t.C {
			update()
			t.Reset(time.Minute)
		}
	}()
}
