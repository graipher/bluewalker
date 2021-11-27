# Bluewalker

Bluewalker is a proof-of-concept Bluetooth LE scanner and advertiser which uses
_HCI user channel_ raw HCI sockets available on Linux to scan for Bluetooth LE
devices or to start advertising with user-specified data. User channel sockets
allow raw access to the Bluetooth controller and bypasses the Linux Bluetooth
stack. This allows Bluewalker access to all Bluetooth LE Advertisement Data sent
by nearby Bluetooth LE devices.

[![pipeline status](https://gitlab.com/jtaimisto/bluewalker/badges/master/pipeline.svg)](https://gitlab.com/jtaimisto/bluewalker/commits/master)
[![coverage report](https://gitlab.com/jtaimisto/bluewalker/badges/master/coverage.svg)](https://gitlab.com/jtaimisto/bluewalker/commits/master)

Bluewalker can be used in four different modes:

- In [**Collector**](#collector-mode) mode it listens for advertising packets
  for given duration and prints summary of the received advertisement packets
  from different devices.
- In [**Observer**](#observer-mode) mode it prints information about received
  advertisement packets as they are received
- In [**Broadcaster**](#broadcaster-mode-sending-advertising-packets) mode
  bluewalker can be used to send advertisement packets
- [**Ruuvi**](#scanning-for-ruuvitag) mode is special mode for listening and
  printing information from Ruuvi tags.
- [**Mijia**](#scanning-for-mijia) mode is special mode for listening and
  printing information from Xiaomi Mijia temperature and humidity devices.

## Installing

### Using go get
On Linux, with go 1.15 or newer installed:

```
$ go get gitlab.com/jtaimisto/bluewalker
```

This will install the binary to ~/bin by default.

### Using docker

If you have docker installed, you can build the bluewalker binary by running
the `build.sh` script. The script will use `Dockerfile` to build and
run docker image which builds bluewalker binary. The resulting binary is
copied to the same directory where the script is run.

By default `build.sh` builds Linux x86_64 binary, desired architecture can
be given to `build.sh` as a parameter. Thus to build Linux ARM binary, for
example for Raspberry Pi devices, run
```
$ ./build.sh arm
```

## Usage

```
Usage of ./bluewalker:
  -active
        Active scanning
  -adv-data string
        Advertising data to send on broadcast mode (Format: "<type>,<data>;<type>,<data>", all values hexadecimal)
  -broadcast
        Send advertising data instead of scanning for it
  -debug
        Enable debug messages
  -device string
        HCI device to use
  -duration int
        Number of seconds to scan, -1 to scan indefinitely (default 5)
  -filter-addata string
        Only show devices whose Advertising Data matches given filter (Format: "<type>,<data>;<type>,<data>", all values hexadecimal)
  -filter-addr string
        List of addresses where advertisement data is accepted from
  -filter-adtype string
        Only show devices whose Advertising data contains structures with specified type(s)
  -filter-irk string
        Only show devices which can be resolved by given IRK
  -filter-partial-addr string
        Filter by partial address bytes
  -filter-vendor string
        Only show devices whose vendor specific advertising data starts with given bytes
  -json
        Output data as json
  -listen-unix string
        Path to socket for listening incoming UNIX socket connections
  -log-trace
        Enable more verbose trace logging in addition to debugging
  -mijia
        Scan and display information about Xiaomi Mijia devices
  -observer
        Do scanning in observer mode (display advertising packets as they are received)
  -output-file string
        Write output to given file, ('-' to indicate stdout)
  -random-addr string
        Random LE Address to set
  -ruuvi
        Scan and display information about found Ruuvi tags
  -scan-resp string
        Scan response data to send on broadcast mode (Format: "<type>,<data>;<type>,<data>", all values hexadecimal)
  -unix string
        Unix socket path where to write results
  -version
        Print version number of the program
```

Bluewalker needs the name of Bluetooth device to use as parameter. Available
Bluetooth devices can be viewed with `hciconfig` command. The selected hci
device needs to be down for Bluewalker to be able to use it. Use
`sudo hciconfig <hcidevice> down` to close selected device. Then Bluewalker can
be started using `sudo bluewalker -device <hcidevice>`. Bluewalker needs to be
run as root (or have `cap_net_admin` capability) to be able to access the raw
HCI device.

### Collector -mode

By default Bluewalker listens for available Bluetooth LE advertisements for 5
seconds and then prints information about found devices and all different
advertisement data it has received from each device. The number of seconds to
scan can be changed with `-duration <seconds>` parameter. If duration is given
as -1, the scanning will continue until user presses `ctrl+c` to terminate it.
By default Bluewalker does only passive scanning, active scanning can be turned
on with `-active` parameter.

After the scanning is complete, information gathered is printed:

```
$ sudo ./bluewalker -device hci0 -active
...
Found 3 devices:
Device 78:b8:98:48:bc:5e,random (RSSI:-88 dBm; last seen Sep 19 23:37:49):
Events: Scannable undirected,Scan Response
Advertising Data Structures:
        Complete 16 Bit Service Class UUID : 0x9ffe
        Service Data : 0x9ffe0242354e5036552d3858546b00000165f38f9e05
        Manufacturer Specific : 0xe0000864ca3e2f84
Device c8:c6:4b:bd:12:10,random (RSSI:-71 dBm; last seen Sep 19 23:37:49):
Events: Non connectable undirected
Advertising Data Structures:
        Flags : 0x04; [00000100] (BR/EDR not supported)
        Manufacturer Specific : 0x990403781402c24c0008000004200b3500000000
        Manufacturer Specific : 0x990403781402c24efffcfff804240b2f00000000
        Manufacturer Specific : 0x990403781402c2510004000004200b3500000000
        Manufacturer Specific : 0x990403781402c2500000000004200b4100000000
        Manufacturer Specific : 0x990403781401c24f0000000004240b3500000000
Device 72:b2:e7:2a:75:87,random (RSSI:-59 dBm; last seen Sep 19 23:37:50):
Events: Connectable undirected,Scan Response
Advertising Data Structures:
        Flags : 0x1a; [00011010] (LE General Discoverable,LE & BR/EDR (controller),LE & BR/EDR (host))
        Manufacturer Specific : 0x4c001005011c221b04
```

The device address (`random` indicating random address), RSSI and timestamp of
last received advertising report followed by list of different advertising
events received. After that all _different_ advertising data structures received
from this device are printed. Additional information is printed for some
advertising structures (Flags are parsed, device name is printed, etc).

### Observer -mode

If `-observer` option is given, then Bluewalker will print information about
received packets as they are received instead of collecting them and printing
summary information.

### Filters

- To display information only about devices with given address, use
  `-filter-addr <address>`. Note that if device is advertising with random
  (private) address, add the address type after comma to the address string.
  Multiple addresses can be given if they are separated by semicolons. For
  example:
  `sudo ./bluewalker -device hci0 -filter-addr "4f:c0:f1:51:4f:22,random;57:68:4b:42:45:0a,random"`

- To display information only about devices whose address start with given
  bytes, use `-filter-partial-addr`. The address part can be given either as
  BD_ADDR (`aa:bb:cc`) or as hex string (`0xaabbcc`)

- To filter devices based on the vendor specific advertising data, use
  `-filter-vendor <data>`, where data is matched against the start of vendor
  specific data (if one exists) in advertisement data. For example, to search
  all advertising apple devices use:
  `sudo ./bluewalker -device hci0 -filter-vendor 0x4c00`

- To filter devices based on the _type_ field of the Advertising Data, use
  `-filter-adtype <type>`, where type is comma -separated list of AD Types to
  allow (in hexadecimal). For example, to search all devices which have device
  name in advertisement data use:
  `sudo ./bluewalker -device hci0 -filter-adtype 0x08,0x09`. See
  https://www.bluetooth.com/specifications/assigned-numbers/generic-access-profile
  for list of allowed AD Types.

- To filter devices based on the Advertising Data present, use
  `-filter-addata <type>,<value>;<type2>,<value2>`, where `type` is AD type
  value in hexadecimal and `value` is bytes the advertising data should start
  with. For example, to show only devices whose Advertising Data contains
  Complete Local Name starting with letter `A` (0x41 in hex) use
  `sudo ./bluewalker -device hci0 -filter-addata 0x09,0x41`. More than one type,
  value pairs can be specified by separating them with `;`. If multiple type,
  value -pairs are given the filter will match only devices whose Advertising
  Data conatains _all_ of the specified elements.

- To filter devices based on IRK (Identity Resolving Key), use
  `-filter-irk <key>`, where _key_ is 128-bit Identity Resolving Key to use to
  resolve resolvable private addresses (see, for example
  [here](https://blog.bluetooth.com/bluetooth-technology-protecting-your-privacy)).
  When filtering by IRK, results are shown only for devices who advertise with
  resolvable private address and whose address can be resolved with given IRK.
  _NOTE_: Bluewalker currently assumes that the key given as parameter contains
  bytes in same order as the IRK's stored by BlueZ in Linux.

If multiple different filters are given, only packets matching _all_ different
filters are show. This applies only to different filters, if multiple addresses
are given for address filter, packets from devices with any of those addresses
are shown. Same applies to multiple AD Types given to AD type filter.

### Scanning for RuuviTags

Bluewalker can be used to scan for [RuuviTags](https://ruuvi.com) running the
default [Sensor Tag Beacon firmware](https://lab.ruuvi.com/ruuvitag-fw/). To
enable listening for RuuviTags, start bluewalker with `-ruuvi` parameter.

Bluewalker is able to decode information when the RuuviTag is running in RAW
mode and the information is encoded with version 3 or 5 of the data (see
[here for RuuviTag data specifications](https://github.com/ruuvi/ruuvi-sensor-protocols))

When run in _ruuvi_ mode, bluewalker will display RuuviTag information whenever
it receives data (no need to use `-observer` option):

```
$ sudo ./bluewalker -device hci0 -ruuvi
Ruuvi device f2:2e:df:eb:8e:99,random (static), Data format:v3 (RSSI:-61 dBm)
        Humidity: 47.00% Temperature: 23.84C Pressure: 99774Pa Battery voltage: 2971mV
        Acceleration X: 0.06G, Y: -0.16G, Z: 1.00G
Ruuvi device c8:c6:4b:bd:12:10,random (static), Data format:v3 (RSSI:-67 dBm)
        Humidity: 60.00% Temperature: 20.02C Pressure: 99748Pa Battery voltage: 2863mV
        Acceleration X: 0.01G, Y: -0.01G, Z: 1.06G
Ruuvi device dc:15:32:fd:71:1f,random (static), Data format:v3 (RSSI:-61 dBm)
        Humidity: 53.00% Temperature: 21.41C Pressure: 99711Pa Battery voltage: 3139mV
        Acceleration X: 0.01G, Y: -0.02G, Z: 1.00G
Ruuvi device fc:d0:22:ad:85:7a,random (static), Data format:v5 (RSSI -29 dBm)
        Humidity: 24.39% Temperature: 23.55C Pressure: 101483Pa Battery voltage: 3151mV
        Acceleration X: -0.07G, Y: 0.03G, Z: 1.01G
        TxPower: 4 dBm, Moves: 85, Seqno: 876
```

### Scanning for Covid-19 Exposure Notifications

Bluewalker can be used to scan for COVID-19 Exposure Notifications sent by
Android or Apple iPhones if they have the contact tracing app (for example
[Koronavilkku](https://koronavilkku.fi) in Finland) installed.

The Exposure Notification Service beacons contain Complete 16-bit Service UUID
(type 0x03) and Service Data - 16 bit UUID (type 0x16) Advertising Structures.
The 16-bit UUID assigned for Exposure Notification Service is 0xfd6f. We can use
bluewalkers `-filter-addata` to show only devices which send advertising data
containing these AD Structures:

```
sudo ./bluewalker -device hci0 -filter-addata "0x03,0x6ffd;0x16,0x6ffd" -observer -duration -1
```

Will show the advertising data as it is being received:

```
Device 31:5b:bf:92:cc:ed,random (non-resolvable private) (RSSI:-89 dBm; last seen Sep  4 22:07:40):
Events:Non connectable undirected
Advertising Data Structures:
        Flags : 0x1a; [00011010](LE General Discoverable,LE & BR/EDR (controller),LE & BR/EDR (host))
        Complete 16 Bit Service Class UUID : 0x6ffd
        Service Data: UUID: 0xfd6f, Exposure Notification
                Proximity Identifier: 0x27049fc9e37031dc6cb8ab1a37b59326, Encrypted Metadata: 0xceb176d2
Device 09:55:39:99:c3:54,random (non-resolvable private) (RSSI:-75 dBm; last seen Sep  4 22:07:40):
Events:Non connectable undirected
Advertising Data Structures:
        Flags : 0x1a; [00011010](LE General Discoverable,LE & BR/EDR (controller),LE & BR/EDR (host))
        Complete 16 Bit Service Class UUID : 0x6ffd
        Service Data: UUID: 0xfd6f, Exposure Notification
                Proximity Identifier: 0x8880340faa038aca857b7ee6d407eb9b, Encrypted Metadata: 0x5235e408
```

### Scanning for Mijia

Bluewalker can be used to scan for
[Xiaomi Mijia LYWSD03MMC](https://pvvx.github.io/ATC_MiThermometer/) running
the [custom firmware](https://github.com/pvvx/ATC_MiThermometer). To enable
listening for mijia, start bluewalker with `-mijia` parameter.

Bluewalker is able to decode information when the Mijia information is encoded
with custom format (see
[here for advertising formats](https://github.com/pvvx/ATC_MiThermometer#bluetooth-advertising-formats))

When run in _mijia_ mode, bluewalker will display Nijia information whenever
it receives data (no need to use `-observer` option):

```
sudo bluewalker -device hci0 -observer -mijia
mijia device a4:c1:38:fa:e3:46, Data format:(RSSI -77 dBm)
	UUID: 0x181a Mac: 46:e3:fa:38:c1:a4
	Temperature: 22.38C Humidity: 30.64%  Battery voltage: 3.065V Battery level: 96%
	Counter: 151 Flags: 5
mijia device a4:c1:38:fa:e3:46, Data format:(RSSI -74 dBm)
	UUID: 0x181a Mac: 46:e3:fa:38:c1:a4
	Temperature: 22.35C Humidity: 30.61%  Battery voltage: 3.065V Battery level: 96%
	Counter: 152 Flags: 5
```

To get the raw data you can also listen to advertisements containing Service
Data for UUID 0x181a:


```
sudo bluewalker -device hci0 -observer -filter-addata 0x16,0x1a18
Device a4:c1:38:fa:e3:46 (RSSI:-82 dBm; last seen Nov 25 08:41:54):
Events:Connectable undirected
Advertising Data Structures:
	Service Data: UUID: 0x181a, Data: 0x46e3fa38c1a4bf088d0df20b5f8805
Device a4:c1:38:fa:e3:46 (RSSI:-87 dBm; last seen Nov 25 08:41:56):
Events:Connectable undirected
Advertising Data Structures:
	Service Data: UUID: 0x181a, Data: 0x46e3fa38c1a4c208a20df20b5f8905
```

### Broadcaster mode - sending advertising packets

Bluewalker can also send advertising packets instead of listening for them. Use
`-broadcast` command line flag to start bluewalker in _broadcaster_ mode and use
`-adv-data` (and, optionally `-scan-resp` to set Scan Response) option to set
the advertising data.

To send advertisements using _random address_, use `-random-addr` -option to set
random address to use.

The advertising data, when given with `-adv-data` (same applies also for Scan
Response data given with `-scan-resp`), needs to be given as one or more
_Advertising Structures_ specifed as one byte `type` field followed by comma
(`,`) and `data` field containing one or more byets (both values should be given
as hexadecimal). Advertising structure definitions should be separated with
semicolon (`;`).

The duration as seconds to advertise can be given with `-duration` flag, with
`-1` specifying that advertising should continue until program is terminated
with `<ctrl> + c` keypress.

For example, to advertise for 30 seconds with AD Flags set to value `0x06` (Le
General Discoverable, BR/EDR not supported) and Complete Local Name set to
"`Bluewalker`":

```
$ sudo ./bluewalker -device hci0 -broadcast -adv-data "0x01,0x06;0x09,0x426c756577616c6b657200" -duration30
Setting advertising data:
        Flags : 0x06
        Complete local name : 0x426c756577616c6b657200
Advertising....Done
```

## JSON output

If `-json` command line option is given, bluewalker will produce JSON encoded
output. This applies to _ruuvi_, _mijia_, _observer_ and _collector_ mode.

When JSON data is written to UNIX socket (`-unix <path>`, `-listen-unix <path>`)
or to a file (`-output-file <path>`), the data is written without any identation
as a string terminated by newline (`\n`). Thus, when parsing JSON data from UNIX
socket or a file when newline is encountered you should have received a well
-formed JSON structure (this applies also to JSON printed to stdout if `-` was
given as a path to `-output-file`).

If none of `-unix`, `-listen-unix` or `-output-file` command line options are
given, data is written to stdout indented.

### Writing JSON output to UNIX socket

If `-unix <path>` command line option is given, the Bluewalker will try to
connect to UNIX socket in given path and writes the JSON output to this socket.
If remote end terminates the connection, Bluewalker stops running as well.

With `-listen-unix <path>` command line option Bluewalker will create listening
UNIX socket on given path and JSON output is written to all connected clients.
With listening UNIX socket, the program is not terminated if all clients are
disconnected.

Note that even if `-json` command line option is not given, specifying UNIX
socket path forces JSON output.

### JSON definitions

When scanning in _collector_ mode, the scan results are presented as JSON array,
where each element on the array represents one device and information gathered
from it during scanning. In _observer_ mode, every received Advertising Data
packet is printed in device information structure as it is received.

The format for device information structure is

```
	{
		"data": [
			{
				"type": 1,
				"data": "BA=="
			},
			{
				"type": 255,
				"data": "mQQDYxJPwZ7/9AAABCALIwAAAAA="
			},
			{
				"type": 255,
				"data": "mQQDYxJOwZr/+P/8BAwLNQAAAAA="
			},
			{
				"type": 255,
				"data": "mQQDYxJQwaX/9AAABBwLIwAAAAA="
			}
		],
		"last": "2018-10-09T17:30:44.100611924+03:00",
		"rssi": -75,
		"types": [
			"Non connectable undirected"
		],
		"device": {
			"address": "c8:c6:4b:bd:12:10",
			"type": "LE Random"
		}
	}
```

| JSON element   | Value                                                                           |
| -------------- | ------------------------------------------------------------------------------- |
| data           | Array of all different advertising data received from the device                |
| data:type      | The AD Type field value (in decimal)                                            |
| data:data      | The received advertising bytes (base64 encoded string)                          |
| last           | timestamp when data was last received from this device                          |
| rssi           | RSSI from the last received packet                                              |
| types          | Array of strings containing names of different advertising event types received |
| device         | Address of the device where the data was received                               |
| device:address | Bluetooth address as string                                                     |
| device:type    | Bluetooth address type (`LE Public`, `LE Random`)                               |

When scanning for ruuvi tags, the information about ruuvi tag is printed as JSON
object every time data is received. Note that _txpower_, _movementCount_ and
_sequence_ are only available if Ruuvi tag is sending with data format 5. If
data is received in format 3, these fields are set to "Not Available" values.

```
{
        "device": {
                "address": "fc:d0:22:ad:85:7a",
                "type": "LE Random"
        },
        "rssi": -40,
        "time": "2021-07-16T12:56:17.319636631+03:00",
        "type": "ruuvi",
        "sensors": {
                "humidity": 23.8375,
                "temperature": 23.449999,
                "pressure": 101494,
                "accelerationX": -0.072,
                "accelerationY": 0.024,
                "accelerationZ": 1.012,
                "voltage": 3085,
                "txpower": 4,
                "movementCount": 109,
                "sequence": 1994
        }
}
```

| JSON element          | Value                                                 |
| --------------------- | ----------------------------------------------------- |
| device                | Address of the Ruuvi tag                              |
| device:address        | Bluetooth address as string                           |
| device:type           | Bluetooth address type (`LE Public`, `LE Random`)     |
| rssi                  | RSSI value from the received advertising event (int)  |
| time                  | Time when the event was received (string)             |
| type                  | Type field indicating type of the data: "ruuvi"       |
| sensors               | Values for all the ruuvi tag sensors                  |
| sensors:humidity      | Humidity value (float)                                |
| sensors:temperature   | Temperature in C (float)                              |
| sensors:pressure      | Pressure in Pa (int)                                  |
| sensors:accelerationX | Acceleration for X axis in G (float)                  |
| sensors:accelerationY | Acceleration for Y axis in G (float)                  |
| sensors:accelerationZ | Acceleration for Y axis in G (float)                  |
| sensors:voltage       | Battery voltage in mV (int)                           |
| sensors:txpower       | TX power level (int, 31 for "Not Available")          |
| sensors:movementCount | Movement counter value (int, 255 for "Not Available") |
| sensors:sequence      | Data sequence number (int, 65535 for "Not Available") |

When scanning for Mijia devices, the information about Mijia device is printed
as JSON object every time data is received. Fields are
[described here](https://github.com/pvvx/ATC_MiThermometer#custom-format-all-data-little-endian)

```
{
	"device": {
		"address": "a4:c1:38:fa:e3:46",
		"type": "LE Public"
	},
	"rssi": -76,
	"time": "2021-11-26T16:40:16.74475812+02:00",
      "type": "mijia",
	"sensors": {
		"uuid": 6170,
		"mac": [
			70,
			227,
			250,
			56,
			193,
			164
		],
		"temperature": 22.29,
		"humidity": 29.75,
		"voltage": 3.046,
		"level": 94,
		"counter": 62,
		"flags": 5
	}
}
```

| JSON element          | Value                                                 |
| --------------------- | ----------------------------------------------------- |
| device                | Address of the Mijia device                           |
| device:address        | Bluetooth address as string                           |
| device:type           | Bluetooth address type (`LE Public`, `LE Random`)     |
| rssi                  | RSSI value from the received advertising event (int)  |
| time                  | Time when the event was received (string)             |
| type                  | Type field indicating type of the data: "mijia"       |
| sensors               | Values for the Mijia information                      |
| sensors:uuid          | UUID, GATT Service 0x181A Environmental Sensing       |
| sensors:temperature   | Temperature in C (float)                              |
| sensors:humidity      | Humidity value (float)                                |
| sensors:voltage       | Battery voltage (float)                               |
| sensors:counter       | Data sequence number (uint8)                          |
| sensors:flags         | Flags for extra info per bit (uint16), in custom mode |

The flags are bits, only available in custom format GPIO_TRG pin (marking
"reset" on circuit board) flags:

| Bit  | Purpose                                                           |
| ---- | ----------------------------------------------------------------- |
| bit0 | Reed Switch, input                                                |
| bit1 | GPIO_TRG pin output value (pull Up/Down)                          |
| bit2 | Output GPIO_TRG pin is controlled according to the set parameters |
| bit3 | Temperature trigger event                                         |
| bit4 | Humidity trigger event                                            |
| bit5 | Protocol doesn't support flags                                    |
