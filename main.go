package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter            = bluetooth.DefaultAdapter
	upliftService      = bluetooth.New16BitUUID(0xfe60) // 0000fe60-0000-1000-8000-00805f9b34fb
	command            = bluetooth.New16BitUUID(0xfe61) // 0000fe61-0000-1000-8000-00805f9b34fb
	response           = bluetooth.New16BitUUID(0xfe62) // 0000fe62-0000-1000-8000-00805f9b34fb
	commandRaise       = []byte{241, 241, 1, 0, 1, 126}
	commandLower       = []byte{241, 241, 2, 0, 2, 126}
	commandGoToPreset1 = []byte{241, 241, 5, 0, 5, 126}
	commandGoToPreset2 = []byte{241, 241, 6, 0, 6, 126}
	commandStatus      = []byte{241, 241, 7, 0, 7, 126} // Gets current height with other unknowns

	// Other commands found from experimentation
	// []byte{241, 241, 0, 0, 0, 126} 	// Wakes up the display but doesn't  do anything
	// []byte{241, 241, 3, 0, 3, 126} 	// Reprogram preset 1 to current desk height
	// []byte{241, 241, 4, 0, 4, 126} 	// Reprogram preset 2 to current desk height
	// []byte{241, 241, 8, 0, 8, 126}   // Returns unknown data
	// []byte{241, 241, 9, 0, 9, 126}   // Returns unknown data
	// []byte{241, 241, 10, 0, 10, 126} // Wakes up the display but doesn't do anything
)

type desk struct {
	// Harware info
	address      bluetooth.Address
	model        string
	manufacturer string
	serialNumber string
	hardwareRev  string
	softwareRev  string
	firmwareRev  string

	// Bluetooth stuff
	command  *bluetooth.DeviceCharacteristic
	response *bluetooth.DeviceCharacteristic

	// Status
	height float32 // in inches
}

type config struct {
	scan          bool
	deviceAddress string
	preset1Time   time.Duration
	preset2Time   time.Duration
}

func (c *config) registerFlagsAndApplyDefaults() {
	flag.BoolVar(&c.scan, "scan", false, "Scan and print nearby devices then exit")
	flag.StringVar(&c.deviceAddress, "device", "", "Chosen device address")
	flag.DurationVar(&c.preset1Time, "preset1", 15*time.Minute, "Time to stay in preset1 position")
	flag.DurationVar(&c.preset2Time, "preset2", 15*time.Minute, "Time to stay in preset2 position")
}

func main() {
	c := &config{}
	c.registerFlagsAndApplyDefaults()
	flag.Parse()

	if !c.scan && c.deviceAddress == "" {
		fmt.Println("Must specify one of --scan or --device")
		return
	}

	err := adapter.Enable()
	if err != nil {
		panic(err)
	}

	if c.scan {
		scan()
		return
	}

	deviceAddr := &bluetooth.Address{}
	deviceAddr.Set(c.deviceAddress)

	fmt.Println("Connecting to:", deviceAddr)
	desk, err := connect(*deviceAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to desk:")
	fmt.Println("  Model       ", desk.model)
	fmt.Println("  Manufacturer", desk.manufacturer)
	fmt.Println("  SerialNumber", desk.serialNumber)
	fmt.Println("  HardwareRev ", desk.hardwareRev)
	fmt.Println("  SoftwareRev ", desk.softwareRev)
	fmt.Println("  FirmwareRev ", desk.firmwareRev)

	listenAndControl(desk, c)
}

func scan() error {
	fmt.Println("Scanning for nearby devices.  Press ctrl-c to exit...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		if device.HasServiceUUID(upliftService) {
			fmt.Println("Found Uplift desk:", "address=", device.Address.String(), "name=", device.LocalName())
		}
	})
	return err
}

func connect(address bluetooth.Address) (*desk, error) {
	desk := &desk{
		address: address,
	}

	device, err := adapter.Connect(address, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, err
	}

	services, err := device.DiscoverServices([]bluetooth.UUID{upliftService, bluetooth.ServiceUUIDDeviceInformation})
	if err != nil {
		return nil, err
	}

	// Uplift desk service
	chars, err := services[0].DiscoverCharacteristics(nil) //[]bluetooth.UUID{command, response})
	if err != nil {
		return nil, err
	}

	for _, char := range chars {
		char := char
		switch char.UUID() {
		case command:
			desk.command = &char
		case response:
			desk.response = &char
		default:
			x, err := readCharacteristic(char)
			fmt.Println("Unknown characteristic", char.UUID(), x, err)
			char.EnableNotifications(func(buf []byte) {
				fmt.Println("Got bytes for char", char.UUID(), buf)
			})
		}
	}

	// Device information service
	chars, err = services[1].DiscoverCharacteristics(nil)
	if err != nil {
		return nil, err
	}

	for _, char := range chars {
		switch char.UUID() {
		case bluetooth.CharacteristicUUIDManufacturerNameString:
			desk.manufacturer, _ = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDHardwareRevisionString:
			desk.hardwareRev, _ = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDModelNumberString:
			desk.model, _ = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDSerialNumberString:
			desk.serialNumber, _ = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDSoftwareRevisionString:
			desk.softwareRev, _ = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDFirmwareRevisionString:
			desk.firmwareRev, _ = readCharacteristic(char)
		}
	}

	return desk, nil
}

func listenAndControl(desk *desk, c *config) error {
	err := desk.response.EnableNotifications(func(buf []byte) {
		if handleResponse(buf, desk) {
			fmt.Println("Desk height:", desk.height)
		}
	})
	if err != nil {
		return nil
	}

	var (
		stdin        = readStdIn()
		dur, nextDur = c.preset1Time, c.preset2Time
		t            = time.NewTimer(dur)
	)

	// fmt.Println("Going to preset 1 for", dur)
	// desk.command.WriteWithoutResponse(commandGoToPreset1)

	fmt.Println("Press ctrl+c to exit or enter command and press enter:")
	fmt.Println("Commands:")
	fmt.Println("  1 - go to preset 1")
	fmt.Println("  2 - go to preset 2")
	fmt.Println("  r - raise the desk a bit")
	fmt.Println("  l - lower the desk a bit")
	for {
		select {
		case <-t.C:
			switch dur {
			case c.preset1Time:
				fmt.Println("Going to preset 2 for", nextDur)
				desk.command.WriteWithoutResponse(commandGoToPreset2)
			case c.preset2Time:
				fmt.Println("Going to preset 1 for", nextDur)
				desk.command.WriteWithoutResponse(commandGoToPreset1)
			}
			dur, nextDur = nextDur, dur
			t.Reset(dur)

		case command := <-stdin:
			switch command {
			case "1":
				desk.command.WriteWithoutResponse(commandGoToPreset1)
			case "2":
				desk.command.WriteWithoutResponse(commandGoToPreset2)
			case "s":
				desk.command.WriteWithoutResponse(commandStatus)
			case "r":
				desk.command.WriteWithoutResponse(commandRaise)
			case "l":
				desk.command.WriteWithoutResponse(commandLower)
			default:
				fmt.Println("Unknown command: '", command, "'")
			}
		}
	}
}

func readStdIn() <-chan string {
	ch := make(chan string)

	go func() {
		for {
			var command string
			_, err := fmt.Scanln(&command)
			if err == nil {
				ch <- command
			}
		}
	}()
	return ch
}

func readCharacteristic(char bluetooth.DeviceCharacteristic) (string, error) {
	buf := make([]byte, 1024)
	n, err := char.Read(buf)
	if err != nil {
		return "", err
	}
	buf = buf[:n]
	return string(buf), nil
}

func handleResponse(buf []byte, desk *desk) (wasHeightChange bool) {
	// Here are examples of the packets when desk
	// changes to various heights (inches) in response to
	// user action (pressing buttons)
	// Bytes 4 and 5 is a 16-value in 0.1 inch increments
	//                   |   |
	//                   v   v
	// 25.2 [242 242 1 3 0 252 7   7 126]
	// 25.3 [242 242 1 3 0 253 7   8 126]
	// 25.5 [242 242 1 3 0 255 7  10 126]
	// 25.6 [242 242 1 3 1   0 7  12 126]
	// 40.3 [242 242 1 3 1 147 7 159 126]
	// 40.5 [242 242 1 3 1 149 7 161 126]
	// 40.7 [242 242 1 3 1 151 7 163 126]

	// When we send command 7 to get the status the
	// packet comes back with one less leading byte.
	// No idea why
	//               |   |
	//               v   v
	// 25.2 [242 1 3 0 252 15 15 126]

	if len(buf) < 8 || len(buf) > 9 || !bytes.HasPrefix(buf, []byte{242}) || !bytes.HasSuffix(buf, []byte{126}) {
		return false
	}

	// There can be 1 or 2 leading bytes. Not sure why.
	buf = bytes.TrimLeft(buf, string([]byte{242}))

	switch {
	case buf[0] == 1 && buf[1] == 3:
		// 1,3 means desk height notification
		h := float32(binary.BigEndian.Uint16(buf[2:4])) / 10
		changed := desk.height != h
		desk.height = h
		return changed
	}

	return false
}
