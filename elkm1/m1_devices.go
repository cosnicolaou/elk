// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package elkm1

import (
	"fmt"
	"log/slog"

	"github.com/cosnicolaou/automation/devices"
)

func NewController(typ string, opts devices.Options) (devices.Controller, error) {
	if typ == "elk-m1xep" {
		return NewM1XEP(opts), nil
	}
	return nil, fmt.Errorf("unsupported elk device type %s", typ)
}

func NewDevice(typ string, opts devices.Options) (devices.Device, error) {
	if typ == "elk-m1zone" {
		return NewZone(opts), nil
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
	m1     *M1xep
	logger *slog.Logger
}

func (d *m1DeviceBase) SetController(c devices.Controller) {
	d.m1 = c.Implementation().(*M1xep)
}

func (d *m1DeviceBase) ControlledBy() devices.Controller {
	return d.m1
}
