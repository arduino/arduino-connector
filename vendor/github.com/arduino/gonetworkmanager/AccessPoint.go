package gonetworkmanager

import (
	"encoding/json"

	"github.com/godbus/dbus"
)

const (
	AccessPointInterface = NetworkManagerInterface + ".AccessPoint"

	AccessPointPropertyFlags      = AccessPointInterface + ".Flags"
	AccessPointPropertyWPAFlags   = AccessPointInterface + ".WpaFlags"
	AccessPointPropertyRSNFlags   = AccessPointInterface + ".RsnFlags"
	AccessPointPropertySSID       = AccessPointInterface + ".Ssid"
	AccessPointPropertyFrequency  = AccessPointInterface + ".Frequency"
	AccessPointPropertyHWAddress  = AccessPointInterface + ".HwAddress"
	AccessPointPropertyMode       = AccessPointInterface + ".Mode"
	AccessPointPropertyMaxBitrate = AccessPointInterface + ".MaxBitrate"
	AccessPointPropertyStrength   = AccessPointInterface + ".Strength"
)

type AccessPoint interface {
	// GetFlags gets flags describing the capabilities of the access point.
	GetFlags() uint32

	// GetWPAFlags gets flags describing the access point's capabilities
	// according to WPA (Wifi Protected Access).
	GetWPAFlags() uint32

	// GetRSNFlags gets flags describing the access point's capabilities
	// according to the RSN (Robust Secure Network) protocol.
	GetRSNFlags() uint32

	// GetSSID returns the Service Set Identifier identifying the access point.
	GetSSID() string

	// GetFrequency gets the radio channel frequency in use by the access point,
	// in MHz.
	GetFrequency() uint32

	// GetHWAddress gets the hardware address (BSSID) of the access point.
	GetHWAddress() string

	// GetMode describes the operating mode of the access point.
	GetMode() Nm80211Mode

	// GetMaxBitrate gets the maximum bitrate this access point is capable of, in
	// kilobits/second (Kb/s).
	GetMaxBitrate() uint32

	// GetStrength gets the current signal quality of the access point, in
	// percent.
	GetStrength() uint8

	GetObjectPath() dbus.ObjectPath

	MarshalJSON() ([]byte, error)
}

func NewAccessPoint(objectPath dbus.ObjectPath) (AccessPoint, error) {
	var a accessPoint
	a.path = objectPath
	return &a, a.init(NetworkManagerInterface, objectPath)
}

type accessPoint struct {
	dbusBase
	path dbus.ObjectPath
}

func (a *accessPoint) GetObjectPath() dbus.ObjectPath {
	return a.path
}

func (a *accessPoint) GetFlags() uint32 {
	return a.getUint32Property(AccessPointPropertyFlags)
}

func (a *accessPoint) GetWPAFlags() uint32 {
	return a.getUint32Property(AccessPointPropertyWPAFlags)
}

func (a *accessPoint) GetRSNFlags() uint32 {
	return a.getUint32Property(AccessPointPropertyRSNFlags)
}

func (a *accessPoint) GetSSID() string {
	return string(a.getSliceByteProperty(AccessPointPropertySSID))
}

func (a *accessPoint) GetFrequency() uint32 {
	return a.getUint32Property(AccessPointPropertyFrequency)
}

func (a *accessPoint) GetHWAddress() string {
	return a.getStringProperty(AccessPointPropertyHWAddress)
}

func (a *accessPoint) GetMode() Nm80211Mode {
	return Nm80211Mode(a.getUint32Property(AccessPointPropertyMode))
}

func (a *accessPoint) GetMaxBitrate() uint32 {
	return a.getUint32Property(AccessPointPropertyMaxBitrate)
}

func (a *accessPoint) GetStrength() uint8 {
	return a.getUint8Property(AccessPointPropertyStrength)
}

func (a *accessPoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"Flags":      a.GetFlags(),
		"WPAFlags":   a.GetWPAFlags(),
		"RSNFlags":   a.GetRSNFlags(),
		"SSID":       a.GetSSID(),
		"Frequency":  a.GetFrequency(),
		"HWAddress":  a.GetHWAddress(),
		"Mode":       a.GetMode().String(),
		"MaxBitrate": a.GetMaxBitrate(),
		"Strength":   a.GetStrength(),
	})
}
