package gonetworkmanager

import (
	"encoding/binary"
	"encoding/json"
	"net"

	"github.com/godbus/dbus"
)

const (
	ConnectionInterface = SettingsInterface + ".Connection"

	ConnectionAddConnection = SettingsInterface + ".AddConnection"
	ConnectionGetSettings   = ConnectionInterface + ".GetSettings"
)

//type ConnectionSettings map[string]map[string]interface{}
type ConnectionSettings map[string]map[string]interface{}

type Connection interface {
	// GetSettings gets the settings maps describing this network configuration.
	// This will never include any secrets required for connection to the
	// network, as those are often protected. Secrets must be requested
	// separately using the GetSecrets() call.
	GetSettings() ConnectionSettings

	AddWirelessConnection(name, password string) dbus.ObjectPath

	MarshalJSON() ([]byte, error)
}

func NewConnection(objectPath dbus.ObjectPath) (Connection, error) {
	var c connection
	return &c, c.init(NetworkManagerInterface, objectPath)
}

type connection struct {
	dbusBase
}

func ip2int(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func (c *connection) AddWirelessConnection(name, password string) dbus.ObjectPath {
	var settings ConnectionSettings
	var ret dbus.ObjectPath

	settings = make(ConnectionSettings)
	settings["802-11-wireless"] = make(map[string]interface{})
	settings["802-11-wireless-security"] = make(map[string]interface{})
	settings["connection"] = make(map[string]interface{})

	settings["802-11-wireless"]["ssid"] = []byte(name)
	settings["802-11-wireless"]["security"] = "802-11-wireless-security"
	settings["802-11-wireless-security"]["psk"] = password
	settings["802-11-wireless-security"]["key-mgmt"] = "wpa-psk"

	settings["connection"]["id"] = name
	settings["connection"]["type"] = "802-11-wireless"

	c.call(&ret, ConnectionAddConnection, settings)
	return ret
}

func (c *connection) GetSettings() ConnectionSettings {
	var settings map[string]map[string]dbus.Variant
	c.call(&settings, ConnectionGetSettings)

	rv := make(ConnectionSettings)

	for k1, v1 := range settings {
		rv[k1] = make(map[string]interface{})

		for k2, v2 := range v1 {
			rv[k1][k2] = v2.Value()
		}
	}

	return rv
}

func (c *connection) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.GetSettings())
}
