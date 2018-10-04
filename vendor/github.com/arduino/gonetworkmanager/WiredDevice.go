package gonetworkmanager

import (
	"encoding/json"

	"github.com/godbus/dbus"
)

const (
	WiredDeviceInterface = DeviceInterface + ".Wired"

	WiredDeviceHWAddress = WiredDeviceInterface + ".HwAddress"
)

type WiredDevice interface {
	Device

	GetHWAddress() string
}

func NewWiredDevice(objectPath dbus.ObjectPath) (WiredDevice, error) {
	var d wiredDevice
	d.path = objectPath
	return &d, d.init(NetworkManagerInterface, objectPath)
}

type wiredDevice struct {
	device
	path dbus.ObjectPath
}

func (d *wiredDevice) GetObjectPath() dbus.ObjectPath {
	return d.path
}

func (d *wiredDevice) GetHWAddress() string {
	return d.getStringProperty(WiredDeviceHWAddress)
}

func (d *wiredDevice) MarshalJSON() ([]byte, error) {
	m := d.device.marshalMap()
	m["HWAddress"] = d.GetHWAddress()
	return json.Marshal(m)
}
