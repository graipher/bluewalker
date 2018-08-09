Bluewalker
---

Bluewalker is a proof-of-concept Bluetooth LE scanner which uses
_HCI user channel_ raw HCI sockets available on Linux to scan for
Bluetooth LE devices. User channel sockets allow raw access to the Bluetooth
controller and bypasses the Linux Bluetooth stack. This allows Bluewalker
access to all Bluetooth LE Advertisement Data sent by nearby Bluetooth LE
devices.

Compilation
---

Bluewalker needs `golang.org/x/sys/unix` package to be installed, it can
be installed with `go get golang.org/x/sys/unix`. After this package is
installed, Bluewalker can be compiled (on linux) with `go install gitlab.com/jtaimisto/bluewalker`  

Usage
---

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

To display information only about devices with given address, use
`-filter-addr <address>`. Note that if device is advertising with private
(random) address, add the address type after comma to the address string. 
Multiple addresses can be given if they are separated by semicolons. 
For example: `sudo ./bluewalker -device hci0 -filter-addr "4f:c0:f1:51:4f:22,private;57:68:4b:42:45:0a,private"`

To filter devices based on the vendor specific advertising data, use 
`-filter-vendor <data>`, where data is matched against the start of vendor
specific data (if one exists) in advertisement data. For example, to search
all advertising apple devices use: `sudo ./bluewalker -device hci0 -filter-vendor 0x4c00` 
