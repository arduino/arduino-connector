// Code generated by "stringer -type=Nm80211Mode"; DO NOT EDIT

package gonetworkmanager

import "fmt"

const _Nm80211Mode_name = "Nm80211ModeUnknownNm80211ModeAdhocNm80211ModeInfraNm80211ModeAp"

var _Nm80211Mode_index = [...]uint8{0, 18, 34, 50, 63}

func (i Nm80211Mode) String() string {
	if i >= Nm80211Mode(len(_Nm80211Mode_index)-1) {
		return fmt.Sprintf("Nm80211Mode(%d)", i)
	}
	return _Nm80211Mode_name[_Nm80211Mode_index[i]:_Nm80211Mode_index[i+1]]
}