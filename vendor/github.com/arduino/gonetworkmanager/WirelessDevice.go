package gonetworkmanager

import (
	"encoding/json"

	"github.com/godbus/dbus"
)

const (
	WirelessDeviceInterface = DeviceInterface + ".Wireless"

	WirelessDeviceGetAccessPoints = WirelessDeviceInterface + ".GetAccessPoints"
	WirelessDeviceRequestScan     = WirelessDeviceInterface + ".RequestScan"
	WirelessDeviceHWAddress       = WirelessDeviceInterface + ".HwAddress"
)

type WirelessDevice interface {
	Device

	// GetAccessPoints gets the list of access points visible to this device.
	// Note that this list does not include access points which hide their SSID.
	// To retrieve a list of all access points (including hidden ones) use the
	// GetAllAccessPoints() method.
	GetAccessPoints() []AccessPoint

	GetHWAddress() string
}

func NewWirelessDevice(objectPath dbus.ObjectPath) (WirelessDevice, error) {
	var d wirelessDevice
	d.path = objectPath
	return &d, d.init(NetworkManagerInterface, objectPath)
}

type wirelessDevice struct {
	device
	path dbus.ObjectPath
}

func (d *wirelessDevice) GetObjectPath() dbus.ObjectPath {
	return d.path
}

func (d *wirelessDevice) GetAccessPoints() []AccessPoint {
	var apPaths []dbus.ObjectPath
	var opts map[string]interface{}

	d.callNoFail(nil, WirelessDeviceRequestScan, opts)

	d.call(&apPaths, WirelessDeviceGetAccessPoints)
	aps := make([]AccessPoint, len(apPaths))

	var err error
	for i, path := range apPaths {
		aps[i], err = NewAccessPoint(path)
		if err != nil {
			panic(err)
		}
	}

	return aps
}

func (d *wirelessDevice) GetHWAddress() string {
	return d.getStringProperty(WirelessDeviceHWAddress)
}

func (d *wirelessDevice) MarshalJSON() ([]byte, error) {
	m := d.device.marshalMap()
	m["AccessPoints"] = d.GetAccessPoints()
	m["HWAddress"] = d.GetHWAddress()
	return json.Marshal(m)
}
