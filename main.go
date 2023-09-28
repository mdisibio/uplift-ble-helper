package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

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
	commandStatus      = []byte{241, 241, 7, 0, 7, 126} // Notify current status including height with other unknowns

	// Other commands found from experimentation
	// zero  = []byte{241, 241, 0, 0, 0, 126} // Wakes up the display but doesn't  do anything
	// preset1Program = []byte{241, 241, 3, 0, 3, 126} // Reprogram preset 1
	// preset2Program = []byte{241, 241, 4, 0, 4, 126} // Reprogram preset 2
	// eight   = []byte{241, 241, 8, 0, 8, 126}   // Notify current status unknowns
	// nine    = []byte{241, 241, 9, 0, 9, 126}   // Notify current status unknowns
	// ten     = []byte{241, 241, 10, 0, 10, 126} // Wakes up the display but doesn't do anything
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

func main() {
	deviceAddr := &bluetooth.Address{}
	deviceAddr.Set("20e46da8-f165-8fbd-39a4-81cb8a91f664")

	// Enable BLE interface.
	err := adapter.Enable()
	if err != nil {
		panic(err)
	}

	// devices := map[string]bluetooth.ScanResult{}

	// Start scanning.
	/*println("scanning...")
	err = adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		// addr := device.Address.String()
		// if prior, ok := devices[addr]; !ok || !bytes.Equal(device.AdvertisementPayload.Bytes(), prior.AdvertisementPayload.Bytes()) {
		//	devices[addr] = device
		if device.Address == *deviceAddr && device.LocalName() != "" {
			fmt.Println("found device:", "Address:", device.Address.String(), "RSSI:", device.RSSI, "LocalName", device.LocalName(), device)

			if device.HasServiceUUID(serviceuuid) {
				fmt.Println("Has service!")
			}

			d, err := adapter.Connect(device.Address, bluetooth.ConnectionParams{})
			if err != nil {
				fmt.Println("Error connecting to device", err)
				return
			}

		}



		services, err := device.DiscoverServices(nil)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("  Found services:")
		for _, s := range services {
			fmt.Println("  ", s.String())
		}*/

	desk := desk{}

	fmt.Println("Connecting")
	device, err := adapter.Connect(*deviceAddr, bluetooth.ConnectionParams{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected")
	desk.address = *deviceAddr

	fmt.Println("Connected")
	services, err := device.DiscoverServices([]bluetooth.UUID{upliftService, bluetooth.ServiceUUIDDeviceInformation})
	if err != nil {
		panic(err)
	}

	// Uplift desk service
	chars, err := services[0].DiscoverCharacteristics(nil)
	if err != nil {
		panic(err)
	}
	for _, char := range chars {
		char := char
		switch char.UUID() {
		case command:
			desk.command = &chars[0]

		case response:
			desk.response = &chars[1]
			desk.response.EnableNotifications(func(buf []byte) {
				if handleResponse(buf, &desk) {
					fmt.Println("Desk height:", desk.height)
				}
			})

		default:
			fmt.Println("Found characteristic: ", char.String(), "Value:", readCharacteristic(char))
		}
	}

	// Device information service
	chars, err = services[1].DiscoverCharacteristics(nil)
	if err != nil {
		panic(err)
	}
	for _, char := range chars {
		// char := char

		switch char.UUID() {
		case bluetooth.CharacteristicUUIDManufacturerNameString:
			desk.manufacturer = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDHardwareRevisionString:
			desk.hardwareRev = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDModelNumberString:
			desk.model = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDSerialNumberString:
			desk.serialNumber = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDSoftwareRevisionString:
			desk.softwareRev = readCharacteristic(char)

		case bluetooth.CharacteristicUUIDFirmwareRevisionString:
			desk.firmwareRev = readCharacteristic(char)
		}
	}

	fmt.Println("Listening to desk:", desk)

	fmt.Println("Press ctrl+c to exit or enter new height and press enter:")
	/*c := make(chan os.Signal, 1)
	<-c*/
	for {
		if desk.command == nil {
			continue
		}
		var command string
		_, err := fmt.Scanln(&command)
		if err != nil {
			fmt.Println("Error scanning input", err)
			continue
		}
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

func readCharacteristic(char bluetooth.DeviceCharacteristic) string {
	buf := make([]byte, 1024)
	n, err := char.Read(buf)
	if err != nil {
		fmt.Println("error reading:", err)
	}
	buf = buf[:n]
	return string(buf)
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
	// packet comes back with one less leading bytes.
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
		// Desk height
		height := binary.BigEndian.Uint16(buf[2:4])
		desk.height = float32(height) / 10
		return true
	}

	return false
}
