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
5 seconds and then prints information about found devices and (all) 
advertisement data it has received. The number of seconds to scan can be
changed with `-duration <seconds>` parameter. By default Bluewalker does only
passive scanning, active scanning can be turned on with `-active` parameter.
