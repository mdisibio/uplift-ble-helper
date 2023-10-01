package main

import (
	"fmt"
	"time"
)

var (
	deskActive       = true
	deskActiveReason = ""
)

func backgroundChecks(c *config) error {
	if c.detectDisplay {
		if err := supportsDisplayCheck(); err != nil {
			return fmt.Errorf("display detection error: %w", err)
		}
		fmt.Println("Enabling auto display detection")
	}

	if c.detectExternalPower {
		if err := supportsExternalPowerCheck(); err != nil {
			return fmt.Errorf("external power detection error: %w", err)
		}
		fmt.Println("Enabling external power source detection")
	}

	update := func() {
		active := true

		if c.detectDisplay {
			awake, err := checkDisplay()
			if err != nil {
				fmt.Println("Error detecting display status:", err)
			}

			if !awake {
				deskActiveReason = "DisplayAsleep"
			}

			active = active && awake
		}

		if c.detectExternalPower {
			connected, err := checkIsConnectedToExternalPower()
			if err != nil {
				fmt.Println("Error detecting external power source:", err)
			}

			if !connected {
				deskActiveReason = "DisconnectedExternalPower"
			}

			active = active && connected
		}

		deskActive = active
		metricsDeskActive.Reset()
		if deskActive {
			metricsDeskActive.WithLabelValues("").Set(1)
		} else {
			metricsDeskActive.WithLabelValues(deskActiveReason).Set(0)
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

	return nil
}
