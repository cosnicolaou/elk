// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package protocol

import (
	"context"
	"fmt"

	"github.com/cosnicolaou/automation/net/streamconn"
)

const NumZones = 208

const (
	DisabledZoneType ZoneDef = iota
	BurglarEntryExit1
	BurglarEntryExit2
	BurglarPerimeterInstant
	BurglarInterior
	BurglarInteriorFollower
	BurglarInteriorNight
	BurglarInteriorNightDelay
	Burglar24Hour
	BurglarBoxTamper
	FireAlarm
	FireVerified
	FireSupervisory
	AuxAlarm1
	AuxAlarm2
	Keyfob
	NonAlarm
	CarbonMonoxide
	EmergencyAlarm
	FreezeAlarm
	GasAlarm
	HeatAlarm
	MedicalAlarm
	PoliceAlarm
	PoliceNoIndication
	WaterAlarm
	KeyMomentaryArmDisarm
	KeyMomentaryArmAway
	KeyMomentaryArmStay
	KeyMomentaryDisarm
	KeyOnOff
	MuteAudibles
	PowerSupervisory
	Temperature
	AnalogZone
	PhoneKey
	IntercomKey
)

var (
	zoneTypeNames = []string{
		"Disabled",
		"Burglar Entry/Exit 1",
		"Burglar Entry/Exit 2",
		"Burglar Perimeter Instant",
		"Burglar Interior",
		"Burglar Interior Follower",
		"Burglar Interior Night",
		"Burglar Interior Night Delay",
		"Burglar 24 Hour",
		"Burglar Box Tamper",
		"Fire Alarm",
		"Fire Verified",
		"Fire Supervisory",
		"Aux Alarm 1",
		"Aux Alarm 2",
		"Keyfob",
		"Non Alarm",
		"Carbon Monoxide",
		"Emergency Alarm",
		"Freeze Alarm",
		"Gas Alarm",
		"Heat Alarm",
		"Medical Alarm",
		"Police Alarm",
		"Police No Indication",
		"Water Alarm",
		"Key Momentary Arm / Disarm",
		"Key Momentary Arm Away",
		"Key Momentary Arm Stay",
		"Key Momentary Disarm",
		"Key On/Off",
		"Mute Audibles",
		"Power Supervisory",
		"Temperature",
		"Analog Zone",
		"Phone Key",
		"Intercom Key",
	}
)

type ZoneDef byte

func (z ZoneDef) String() string {
	if int(z) >= len(zoneTypeNames) {
		return fmt.Sprintf("UnknownZoneType(%v)", int(z))
	}
	return zoneTypeNames[z]
}

type ZoneDefs [NumZones]ZoneDef

func GetZoneDefinitions(ctx context.Context, sess streamconn.Session) (ZoneDefs, error) {
	req, resp := request.ZoneDefinitions()
	data, err := rpc(ctx, sess, req, resp)
	if err != nil {
		return ZoneDefs{}, err
	}
	var defs ZoneDefs
	if len(data) != NumZones {
		return ZoneDefs{}, fmt.Errorf("unexpected number of zones: got %v, expected %v", len(data), NumZones)
	}
	for i := range data {
		defs[i] = ZoneDef(data[i] - '0')
	}
	return defs, nil
}

func GetZoneName(ctx context.Context, sess streamconn.Session, zone int) (string, error) {
	req, resp := request.ZoneName(zone)
	data, err := rpc(ctx, sess, req, resp)
	if err != nil {
		return "", err
	}
	zn, name, err := ParseTextDescription(data)
	if err != nil {
		return "", err
	}
	if zn != zone {
		return "", fmt.Errorf("unexpected zone: got %v, expected %v", zn, zone)
	}
	return name, nil
}

type ZonePhysicalStatus byte

const (
	ZoneUnconfigured ZonePhysicalStatus = iota
	ZoneOpen
	ZoneEOL
	ZoneShort
)

var (
	zonePhysicalStatusNames = []string{
		"Unconfigured",
		"Open",
		"EOL",
		"Short",
	}
)

type ZoneLogicalStatus byte

const (
	ZoneNormal ZoneLogicalStatus = iota
	ZoneTrouble
	ZoneViolated
	ZoneBypassed
)

var (
	zoneLogicalStatusNames = []string{
		"Normal",
		"Trouble",
		"Violated",
		"Bypassed",
	}
)

type ZoneStatus byte

func (z ZoneStatus) String() string {
	return fmt.Sprintf("%v/%v", zonePhysicalStatusNames[z&0x3], zoneLogicalStatusNames[(z>>2)&0x3])
}

func (z ZoneStatus) Physical() ZonePhysicalStatus {
	return ZonePhysicalStatus(z & 0x3)
}

func (z ZoneStatus) Logical() ZoneLogicalStatus {
	return ZoneLogicalStatus((z >> 2) & 0x3)
}

type ZoneStatusAll [NumZones]ZoneStatus

// GetZoneStatusAll returns the status of all zones, it should not be used for
// polling.
func GetZoneStatusAll(ctx context.Context, sess streamconn.Session) (ZoneStatusAll, error) {
	req, resp := request.ZoneStatus()
	data, err := rpc(ctx, sess, req, resp)
	if err != nil {
		return ZoneStatusAll{}, err
	}
	return ParseZoneStatus(data)
}
