package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"tinygo.org/x/bluetooth"
)

var (
	adapter            = bluetooth.DefaultAdapter
	upliftService      = bluetooth.New16BitUUID(0xfe60) // 0000fe60-0000-1000-8000-00805f9b34fb
	command            = bluetooth.New16BitUUID(0xfe61) // 0000fe61-0000-1000-8000-00805f9b34fb
	response           = bluetooth.New16BitUUID(0xfe62) // 0000fe62-0000-1000-8000-00805f9b34fb
	commandRaise       = makeCommand(1)                 // []byte{241, 241, 1, 0, 1, 126}
	commandLower       = makeCommand(2)                 // []byte{241, 241, 2, 0, 2, 126}
	commandGoToPreset1 = makeCommand(5)                 // []byte{241, 241, 5, 0, 5, 126}
	commandGoToPreset2 = makeCommand(6)                 // []byte{241, 241, 6, 0, 6, 126}
	commandStatus      = makeCommand(7)                 // []byte{241, 241, 7, 0, 7, 126}

	// Other commands found from experimentation
	// []byte{241, 241,   3, 0,   3, 126} // Reprogram preset 1 to current desk height
	// []byte{241, 241,   4, 0,   4, 126} // Reprogram preset 2 to current desk height
	// []byte{241, 241,   8, 0,   8, 126} // Returns unknown data
	// []byte{241, 241,   9, 0,   9, 126} // Returns unknown data
	// []byte{241, 241,  12, 0,  12, 126} // Returns unknown data
	// []byte{241, 241,  28, 0,  28, 126} // Returns unknown data
	// []byte{241, 241,  31, 0,  31, 126} // Returns unknown data
	// []byte{241, 241,  32, 0,  32, 126} // Returns unknown data
	// []byte{241, 241,  33, 0,  33, 126} // Returns unknown data
	// []byte{241, 241,  34, 0,  34, 126} // Returns unknown data
	// []byte{241, 241,  35, 0,  35, 126} // Returns unknown data
	// []byte{241, 241, 254, 0, 254, 126} // Returns unknown data
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
	height  float32 // in inches
	preset1 float32
	preset2 float32
	preset3 float32
	preset4 float32
}

func (d *desk) setHeight(inches float32) {
	meters := inches * 0.0254
	d.height = inches

	metricsDeskHeight.WithLabelValues(d.address.String()).Set(float64(inches))
	metricsDeskHeightMeters.WithLabelValues(d.address.String()).Set(float64(meters))
}

type config struct {
	scan          bool
	deviceAddress string
	preset1Time   time.Duration
	preset2Time   time.Duration
	port          int
	detectDisplay bool
}

func (c *config) registerFlagsAndApplyDefaults() {
	flag.BoolVar(&c.scan, "scan", false, "Scan and print nearby devices")
	flag.StringVar(&c.deviceAddress, "device", "", "Chosen device address")
	flag.DurationVar(&c.preset1Time, "preset1", 0, "Time to stay in preset1 position")
	flag.DurationVar(&c.preset2Time, "preset2", 0, "Time to stay in preset2 position")
	flag.IntVar(&c.port, "port", 0, "Port to serve http metrics and api")
	flag.BoolVar(&c.detectDisplay, "detect-display", false, "Detect display status and don't automatically move desk when display is asleep")
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
	fmt.Println("  Model         ", desk.model)
	fmt.Println("  Manufacturer  ", desk.manufacturer)
	fmt.Println("  SerialNumber  ", desk.serialNumber)
	fmt.Println("  HardwareRev   ", desk.hardwareRev)
	fmt.Println("  SoftwareRev   ", desk.softwareRev)
	fmt.Println("  FirmwareRev   ", desk.firmwareRev)
	fmt.Println("  Current height", desk.height, "in")
	fmt.Println("  Preset 1      ", desk.preset1, "in")
	fmt.Println("  Preset 2      ", desk.preset2, "in")
	fmt.Println("  Preset 3      ", desk.preset3, "in")
	fmt.Println("  Preset 4      ", desk.preset4, "in")

	if c.port > 0 {
		http.Handle("/metrics", promhttp.Handler())
		addr := fmt.Sprintf(":%d", c.port)
		fmt.Println("Serving http at", addr)
		go func() {
			http.ListenAndServe(addr, nil)
		}()
	}

	if c.detectDisplay {
		if err := supportsDisplayCheck(); err != nil {
			fmt.Println("Display detection error:", err)
		}
		monitorDisplay()
	}

	if c.preset1Time > 0 && c.preset2Time > 0 {
		auto(desk, c)
	}

	cli(desk, c)
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
	chars, err := services[0].DiscoverCharacteristics(nil)
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
			err := desk.response.EnableNotifications(func(buf []byte) {
				if handleResponse(buf, desk) {
					fmt.Println("Desk height:", desk.height)
				}
			})
			if err != nil {
				return nil, err
			}
			desk.command.WriteWithoutResponse(commandStatus)
			time.Sleep(time.Second) // Wait for response
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

func auto(desk *desk, c *config) {
	go func() {
		var (
			dur     = c.preset1Time
			nextDur = c.preset2Time
			t       = time.NewTimer(dur)
		)

		if !c.detectDisplay || awake {
			// Initialize first position
			fmt.Println("Enabling auto desk control.")
			fmt.Println("Going to preset 1 for", dur)
			desk.command.WriteWithoutResponse(commandGoToPreset1)
		}

		for range t.C {
			if c.detectDisplay && !awake {
				fmt.Println("Skipping desk movement because display is not awake")
				t.Reset(dur)
				continue
			}

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
		}
	}()
}

func cli(desk *desk, c *config) {
	stdin := readStdIn()

	fmt.Println("Press ctrl+c to exit or enter command and press enter:")
	fmt.Println("Commands:")
	fmt.Println("  1 - go to preset 1")
	fmt.Println("  2 - go to preset 2")
	fmt.Println("  r - raise the desk a bit")
	fmt.Println("  l - lower the desk a bit")

	for command := range stdin {
		switch command {
		case "1":
			desk.command.WriteWithoutResponse(commandGoToPreset1)
		case "2":
			desk.command.WriteWithoutResponse(commandGoToPreset2)
		case "r":
			desk.command.WriteWithoutResponse(commandRaise)
		case "l":
			desk.command.WriteWithoutResponse(commandLower)
		default:
			// Enter raw commands in the form of comma-delimited decimal notations
			// 241,241,5,0,5,126
			ints := strings.Split(command, ",")
			bytes := []byte{}
			var err error
			var ii int64
			for _, i := range ints {
				ii, err = strconv.ParseInt(i, 0, 16)
				if err != nil {
					fmt.Println(err)
					break
				}
				bytes = append(bytes, byte(ii))
			}
			if err != nil {
				break
			}
			fmt.Println("Sending raw command:", bytes)
			desk.command.WriteWithoutResponse(bytes)
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
	command, params := decodeResponse(buf)
	switch command {
	case 1:
		// Command 1 is desk height report
		// Here are examples of the packets when desk
		// changes to various heights (inches) in response to
		// user action (pressing buttons)
		// The first two param bytes are a big-endian u16 in 0.1 inch increments
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
		if len(params) >= 2 {
			h := decodeHeight(params)
			changed := desk.height != 0 && desk.height != h // Ignore first time
			desk.setHeight(h)
			return changed
		}
	case 37:
		if len(params) == 2 {
			desk.preset1 = decodeHeight(params)
		}
	case 38:
		if len(params) == 2 {
			desk.preset2 = decodeHeight(params)
		}
	case 39:
		if len(params) == 2 {
			desk.preset3 = decodeHeight(params)
		}
	case 40:
		if len(params) == 2 {
			desk.preset4 = decodeHeight(params)
		}
	default:
		fmt.Println("Unknown response:", command, params)
	}

	return false
}

// makeCommand generates the byte sequence for the given command
// and parameters.  The format of the sequence is:
// <marker> <marker> <command> <# of params> <params> <checksum> <marker>
func makeCommand(c byte, params ...byte) []byte {
	checksum := c
	checksum += byte(len(params))
	for _, p := range params {
		checksum += p
	}

	var buf []byte
	buf = append(buf, 241)
	buf = append(buf, 241)
	buf = append(buf, c)
	buf = append(buf, byte(len(params)))
	buf = append(buf, params...)
	buf = append(buf, checksum)
	buf = append(buf, 126)

	return buf
}

func decodeResponse(buf []byte) (command byte, params []byte) {
	if bytes.HasPrefix(buf, []byte{242}) && bytes.HasSuffix(buf, []byte{126}) {
		// TODO verify checksum and param len
		// Handle variable number of leading bytes
		for buf[0] == 242 {
			buf = buf[1:]
		}
		if len(buf) >= 2 {
			// At least 1 command byte and 1 params len byte
			if buf[1] > 0 {
				return buf[0], buf[2 : 2+buf[1]]
			}
			return buf[0], nil
		}
	}

	return 0, nil
}

func decodeHeight(buf []byte) float32 {
	return float32(binary.BigEndian.Uint16(buf[0:2])) / 10
}
