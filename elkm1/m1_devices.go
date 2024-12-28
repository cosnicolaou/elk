// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package elkm1

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/cosnicolaou/automation/devices"
)

func NewController(typ string, opts devices.Options) (devices.Controller, error) {
	switch typ {
	case "elk-m1xep":
		return NewM1XEP(opts), nil
	}
	return nil, fmt.Errorf("unsupported elk device type %s", typ)
}

func NewDevice(typ string, opts devices.Options) (devices.Device, error) {
	switch typ {
	case "elk-m1zone":
		return &Zone{
			logger: opts.Logger.With(
				"protocol", "elk-m1xep",
				"device", "elk-m1zone")}, nil
	}
	return nil, fmt.Errorf("unsupported elk m1 device type %s", typ)
}

func SupportedDevices() devices.SupportedDevices {
	return devices.SupportedDevices{
		"elk-m1zone": NewDevice,
	}
}

func SupportedControllers() devices.SupportedControllers {
	return devices.SupportedControllers{
		"elk-m1xep": NewController,
	}
}

type m1DeviceBase struct {
	devices.DeviceConfigCommon
	m1     *M1xep
	logger *slog.Logger
}

func (d *m1DeviceBase) SetConfig(c devices.DeviceConfigCommon) {
	d.DeviceConfigCommon = c
}

func (d *m1DeviceBase) Config() devices.DeviceConfigCommon {
	return d.DeviceConfigCommon
}

func (d *m1DeviceBase) SetController(c devices.Controller) {
	d.m1 = c.Implementation().(*M1xep)
}

func (d *m1DeviceBase) ControlledByName() string {
	return d.Controller
}

func (d *m1DeviceBase) ControlledBy() devices.Controller {
	return d.m1
}

func (d *m1DeviceBase) Timeout() time.Duration {
	return time.Minute
}

func (d *m1DeviceBase) Conditions() map[string]devices.Condition {
	return map[string]devices.Condition{}
}

func (d *m1DeviceBase) ConditionsHelp() map[string]string {
	return map[string]string{}
}
