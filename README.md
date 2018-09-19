# Bluewalker

Bluewalker is a proof-of-concept Bluetooth LE scanner which uses
_HCI user channel_ raw HCI sockets available on Linux to scan for
Bluetooth LE devices. User channel sockets allow raw access to the Bluetooth
controller and bypasses the Linux Bluetooth stack. This allows Bluewalker
access to all Bluetooth LE Advertisement Data sent by nearby Bluetooth LE
devices.

## Compilation

Bluewalker needs `golang.org/x/sys/unix` package to be installed, it can
be installed with `go get golang.org/x/sys/unix`. After this package is
installed, Bluewalker can be compiled (on linux) with `go install gitlab.com/jtaimisto/bluewalker`  

## Usage

Bluewalker needs the name of Bluetooth device to use as parameter. Available
Bluetooth devices can be viewed with `hciconfig` command. The selected
hci device needs to be down for Bluewalker to be able to use it. Use
`sudo hciconfig <hcidevice> down` to close selected device. Then Bluewalker
can be started using `sudo bluewalker -device <hcidevice>`. Bluewalker needs
to be run as root to be able to access the raw HCI device.

By default Bluewalker listens for avaibale Bluetooth LE advertisements for
5 seconds and then prints information about found devices and all different
advertisement data it has received from each device. The number of seconds to
scan can be changed with `-duration <seconds>` parameter. By default Bluewalker
does only passive scanning, active scanning can be turned on with `-active`
parameter.

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
last received advertising report followed by list of different
advertising events received. After that all _different_ advertising data
structures received from this device are printed.
Additional information is printed for
some advertising structures (Flags are parsed, device name is printed, etc).

### Filters

To display information only about devices with given address, use
`-filter-addr <address>`. Note that if device is advertising with random
(private) address, add the address type after comma to the address string. 
Multiple addresses can be given if they are separated by semicolons. 
For example: `sudo ./bluewalker -device hci0 -filter-addr "4f:c0:f1:51:4f:22,random;57:68:4b:42:45:0a,random"`

To filter devices based on the vendor specific advertising data, use 
`-filter-vendor <data>`, where data is matched against the start of vendor
specific data (if one exists) in advertisement data. For example, to search
all advertising apple devices use: `sudo ./bluewalker -device hci0 -filter-vendor 0x4c00` 

To filter devices based on the _type_ field of the Advertising Data, use
`-filter-adtype <type>`, where type is comma -separated list of AD Types to allow (in hexadecimal).
For example, to search all devices which have device name in advertisement data use:
`sudo ./bluewalker -device hci0 -filter-adtype 0x08,0x09`. See https://www.bluetooth.com/specifications/assigned-numbers/generic-access-profile for list of allowed AD Types. 

### Scanning for RuuviTags

Bluewalker can be used to scan for [RuuviTags](https://ruuvi.com) running the
default [Sensor Tag Beacon firmware](https://lab.ruuvi.com/ruuvitag-fw/).
To enable listening for RuuviTags, start bluewalker with `-ruuvi` parameter.

Bluewalker is able to decode information when the RuuviTag is running in
RAW mode and the information is encoded with version 3 of the data (see
[here for RuuviTag data specifications](https://github.com/ruuvi/ruuvi-sensor-protocols))

When run in *ruuvi* mode, bluewalker will display RuuviTag information whenever
it receives data:
```
$ sudo ./bluewalker -device hci0 -ruuvi
Ruuvi device f2:2e:df:eb:8e:99,random (RSSI:-61 dBm)
        Humidity: 47.00% Temperature: 23.84C Pressure: 99774Pa Battery voltage: 2971mV
        Acceleration X: 0.06G, Y: -0.16G, Z: 1.00G
Ruuvi device c8:c6:4b:bd:12:10,random (RSSI:-67 dBm)
        Humidity: 60.00% Temperature: 20.02C Pressure: 99748Pa Battery voltage: 2863mV
        Acceleration X: 0.01G, Y: -0.01G, Z: 1.06G
Ruuvi device dc:15:32:fd:71:1f,random (RSSI:-61 dBm)
        Humidity: 53.00% Temperature: 21.41C Pressure: 99711Pa Battery voltage: 3139mV
        Acceleration X: 0.01G, Y: -0.02G, Z: 1.00G
```