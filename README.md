# uplift-ble-helper

I have an Uplift standing desk that I absolutely love but wasn't using to its full potential.  This app encourages [healthy](https://pubmed.ncbi.nlm.nih.gov/33813968/) [working](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8582919/) [habits](https://pubmed.ncbi.nlm.nih.gov/33137789/) by automatically raising and lowering the desk throughout the day.  It interfaces through the new [Bluetooth BLE adapter](https://www.upliftdesk.com/bluetooth-adapter-for-uplift-desk/) (FRM125) and also exports Prometheus-style metrics so you can track your habits over time.

Frustration-free:  This app tries to be as unobtrusive as possible and pauses desk movement when the display is asleep or a laptop is disconnected from external power.

## Platform Support

Feature | OSX | Linux | Windows
--------| ----| ------| -------
Automatic desk control | ✅ | ✅ | ✅
Prometheus metrics | ✅| ✅ | ✅
Display detection | ✅ (via the [brightness](https://github.com/nriley/brightness) command | ❌ | ❌
External power detection | ✅ (via `ioreg` which is built-in) | ❌ | ❌

## Getting Started

### Build
Bluetooth, display and power detection means it must run as a native app and not a docker container.  Building it requires Go 1.19 or higher.

```
git clone git@github.com:mdisibio/uplift-ble-helper.git
cd uplift-ble-helper
go build
```

### Get your desk's bluetooth address

Scan for nearby bluetooth adapters with the `--scan` option.
```
./uplift-ble-helper --scan
```

It will print nearby devices. When you find your device's address copy it for the next step.  Press ctrl+c to exit. Note - Only prints devices advertising the Uplift desk service described in __Technical Notes__ below.

```
$ ./uplift-ble-helper --scan
Scanning for nearby devices.  Press ctrl-c to exit...
Found Uplift desk: address= 20e46da8-f165-8fbd-39a4-81cb8a91f664 name= 
Found Uplift desk: address= 20e46da8-f165-8fbd-39a4-81cb8a91f664 name= BLE Device 547278
Found Uplift desk: address= 20e46da8-f165-8fbd-39a4-81cb8a91f664 name= BLE Device 547278
Found Uplift desk: address= 20e46da8-f165-8fbd-39a4-81cb8a91f664 name= BLE Device 547278
Found Uplift desk: address= 20e46da8-f165-8fbd-39a4-81cb8a91f664 name= BLE Device 547278
Found Uplift desk: address= 20e46da8-f165-8fbd-39a4-81cb8a91f664 name= BLE Device 547278
```

### Connect
Use the `--device` option to connect to your device and start standing more!  Out of the box functionality includes sensible defaults but can be totally customized.  See the full command line reference below.  There is also an interactive shell to check functionality and experiment.

```
$ ./uplift-ble-helper --device 20e46da8-f165-8fbd-39a4-81cb8a91f664
Connecting to: 20e46da8-f165-8fbd-39a4-81cb8a91f664
Connected to desk:
  Model          LSD4BT-E95ALSP001
  Manufacturer   Manufacturer Name
  SerialNumber   Serial Number
  HardwareRev    Hardware Revision
  SoftwareRev    v1.13.Dec 14 202
  FirmwareRev    Rev13
  Current height 40.5 in
  Preset 1       40.5 in
  Preset 2       25.2 in
  Preset 3       30.3 in
  Preset 4       25.2 in
Serving http at :9090
Enabling auto display detection
Enabling external power source detection
Enabling auto desk control.
Press ctrl+c to exit or enter command and press enter:
Commands:
  1 - go to preset 1
  2 - go to preset 2
  r - raise the desk a bit
  l - lower the desk a bit
Going to preset 1 for 40m0s
```

## Command line reference

```
./uplift-ble-helper -h
```

### Options
Option | Description | Default
--------| ----| ------
--scan | Scan for nearby devices. |
--device <addr> | Connect to this device. This is the usual mode of operation.  |
--auto | Automatically raise and lower the desk.  Can be disabled with --auto=false | true
--preset1 | The time to spend at preset 1. Uses go duration format like 30m15s | 40m (sitting)
--preset2 | The time to spend at preset 2. Uses go duration format like 30m15s | 20m (standing)
--port | Export prometheus /metrics on this port. Can be disabled with --port=0 | 9090
--detect-display | Pause desk movement when the display is asleep. Can be disabled with --detect-display=false | true
--detect-external-power | This option is for laptops. Pause desk movement if machine is disconnected from external power.  Can be disabled with --detect-external-power=false | true 

## Technical Notes

The Uplift BLE Adapter shares a similar inner UART protocol to others such as Jarvis, and the work to reverse engineer the protocol is generally reusable.  However there are a couple BLE adapters and the exposed BLE services seem to be different. [My dongle](https://www.upliftdesk.com/bluetooth-adapter-for-uplift-desk/) advertises the following:

ID | Type | Description
----- | -- | ------
0000fe60-0000-1000-8000-00805f9b34fb | Service | Main service. Seems to indicate a [Lierda](https://gist.github.com/ariccio/2882a435c79da28ba6035a14c5c65f22#file-bluetoothconstants-ts-L517) chipset, which is corroborated by the model number LSD4BT-E95ALSP001 reported on startup.
0000fe61-0000-1000-8000-00805f9b34fb | Characteristic | Command, we write UART packets to this
0000fe62-0000-1000-8000-00805f9b34fb | Characteristic | Response, we read UART packets from this
0000fe63-0000-1000-8000-00805f9b34fb | Characteristic | Unknown purpose
0000fe62-0000-1000-8000-00805f9b34fb | Characteristic | Unknown purpose

And the following generic device information:

ID | Type | Description
----- | -- | ------
0000180a-0000-1000-8000-00805f9b34fb | Service | Device information
00002a24-0000-1000-8000-00805f9b34fb | Characteristic | Model number
00002a25-0000-1000-8000-00805f9b34fb | Characteristic | Serial number
00002a26-0000-1000-8000-00805f9b34fb | Characteristic | Firmware revision
00002a27-0000-1000-8000-00805f9b34fb | Characteristic | Hardware revisision
00002a28-0000-1000-8000-00805f9b34fb | Characteristic | Software revision
00002a29-0000-1000-8000-00805f9b34fb | Characteristic | Manufacturer name

## Future Improvements

* Linux and Windows support (Help needed!) Core bluetooth and prometheus functionality should be cross-platform, but need help finding and testing ways to detect the display and external power statuses for each OS. 
* Detect machine locked status. This is a bit more precise than display asleep.
* Dashboard (coming soon)
* Reverse engineer fe63 and fe64.  Especially would like to read the device friendly name like "My Desk"

## Acknowledgements

This app would not be possible without the hard work of these projects:
* https://github.com/tinygo-org/bluetooth - Cross platform BLE golang
* https://github.com/hairyhenderson/jarvis_exporter - Original inspiration, awesome dashboard, and showing me the `brightness`
* https://github.com/phord/Jarvis - Reverse engineer of the BLE UART protocol
* https://github.com/justintout/uplift-reconnect - Reverse engineer of the BLE UART protocol
* https://github.com/william-r-s/desk_controller_uplift - Reverse engineer of the BLE UART protocol