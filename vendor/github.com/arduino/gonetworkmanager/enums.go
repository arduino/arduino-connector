package gonetworkmanager

//go:generate stringer -type=NmConnectivity
type NmConnectivity uint32

const (
	NmConnectivityUnknown NmConnectivity = 0
	NmConnectivityNone    NmConnectivity = 1
	NmConnectivityPortal  NmConnectivity = 2
	NmConnectivityLimited NmConnectivity = 3
	NmConnectivityFull    NmConnectivity = 4
)

//go:generate stringer -type=NmState
type NmState uint32

const (
	NmStateUnknown         NmState = 0
	NmStateAsleep          NmState = 10
	NmStateDisconnected    NmState = 20
	NmStateDisconnecting   NmState = 30
	NmStateConnecting      NmState = 40
	NmStateConnectedLocal  NmState = 50
	NmStateConnectedSite   NmState = 60
	NmStateConnectedGlobal NmState = 70
)

//go:generate stringer -type=NmDeviceState
type NmDeviceState uint32

const (
	NmDeviceStateUnknown      NmDeviceState = 0
	NmDeviceStateUnmanaged    NmDeviceState = 10
	NmDeviceStateUnavailable  NmDeviceState = 20
	NmDeviceStateDisconnected NmDeviceState = 30
	NmDeviceStatePrepare      NmDeviceState = 40
	NmDeviceStateConfig       NmDeviceState = 50
	NmDeviceStateNeed_auth    NmDeviceState = 60
	NmDeviceStateIp_config    NmDeviceState = 70
	NmDeviceStateIp_check     NmDeviceState = 80
	NmDeviceStateSecondaries  NmDeviceState = 90
	NmDeviceStateActivated    NmDeviceState = 100
	NmDeviceStateDeactivating NmDeviceState = 110
	NmDeviceStateFailed       NmDeviceState = 120
)

//go:generate stringer -type=NmDeviceType
type NmDeviceType uint32

const (
	NmDeviceTypeUnknown    NmDeviceType = 0
	NmDeviceTypeEthernet   NmDeviceType = 1
	NmDeviceTypeWifi       NmDeviceType = 2
	NmDeviceTypeUnused1    NmDeviceType = 3
	NmDeviceTypeUnused2    NmDeviceType = 4
	NmDeviceTypeBt         NmDeviceType = 5
	NmDeviceTypeOlpcMesh   NmDeviceType = 6
	NmDeviceTypeWimax      NmDeviceType = 7
	NmDeviceTypeModem      NmDeviceType = 8
	NmDeviceTypeInfiniband NmDeviceType = 9
	NmDeviceTypeBond       NmDeviceType = 10
	NmDeviceTypeVlan       NmDeviceType = 11
	NmDeviceTypeAdsl       NmDeviceType = 12
	NmDeviceTypeBridge     NmDeviceType = 13
	NmDeviceTypeGeneric    NmDeviceType = 14
	NmDeviceTypeTeam       NmDeviceType = 15
)

//go:generate stringer -type=Nm80211APFlags
type Nm80211APFlags uint32

const (
	Nm80211APFlagsNone    Nm80211APFlags = 0x0
	Nm80211APFlagsPrivacy Nm80211APFlags = 0x1
)

//go:generate stringer -type=Nm80211APSec
type Nm80211APSec uint32

const (
	Nm80211APSecNone         Nm80211APSec = 0x0
	Nm80211APSecPairWEP40    Nm80211APSec = 0x1
	Nm80211APSecPairWEP104   Nm80211APSec = 0x2
	Nm80211APSecPairTKIP     Nm80211APSec = 0x4
	Nm80211APSecPairCCMP     Nm80211APSec = 0x8
	Nm80211APSecGroupWEP40   Nm80211APSec = 0x10
	Nm80211APSecGroupWEP104  Nm80211APSec = 0x20
	Nm80211APSecGroupTKIP    Nm80211APSec = 0x40
	Nm80211APSecGroupCCMP    Nm80211APSec = 0x80
	Nm80211APSecKeyMgmtPSK   Nm80211APSec = 0x100
	Nm80211APSecKeyMgmt8021X Nm80211APSec = 0x200
)

//go:generate stringer -type=Nm80211Mode
type Nm80211Mode uint32

const (
	Nm80211ModeUnknown Nm80211Mode = 0
	Nm80211ModeAdhoc   Nm80211Mode = 1
	Nm80211ModeInfra   Nm80211Mode = 2
	Nm80211ModeAp      Nm80211Mode = 3
)
