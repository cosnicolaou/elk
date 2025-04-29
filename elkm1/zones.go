// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package elkm1

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosnicolaou/automation/devices"
	"github.com/cosnicolaou/elk/elkm1/protocol"
	"gopkg.in/yaml.v3"
)

type ZoneConfig struct {
	ZoneNumber int `yaml:"zone"`
}

type Zone struct {
	m1DeviceBase
	devices.DeviceBase[ZoneConfig]
}

func (z *Zone) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&z.DeviceConfigCustom); err != nil {
		return err
	}
	if zn := z.DeviceConfigCustom.ZoneNumber; zn > protocol.NumZones {
		return fmt.Errorf("invalid zone number: %v", zn)
	}
	return nil
}

func (z *Zone) Conditions() map[string]devices.Condition {
	return map[string]devices.Condition{
		"normal":   z.Normal,
		"closed":   z.Normal,
		"open":     z.Violated,
		"violated": z.Violated,
		"trouble":  z.Trouble,
		"bypassed": z.Bypassed,
	}
}

func (z *Zone) ConditionsHelp() map[string]string {
	return map[string]string{
		"normal":   "true if the zone is in a normal state",
		"closed":   "true if the zone is in a normal state",
		"open":     "true if the zone is in a violated state",
		"violated": "true if the zone is in a violated state",
		"trouble":  "true if the zone is in a trouble state",
		"bypassed": "true if the zone is in a bypassed state",
	}
}

func NewZone(_ devices.Options) *Zone {
	return &Zone{
		m1DeviceBase: m1DeviceBase{},
	}
}

func (z *Zone) logical(ctx context.Context, opts devices.OperationArgs) (protocol.ZoneStatus, error) {
	ctx, sess := z.m1.Session(ctx)
	status, err := protocol.GetZoneStatusAll(ctx, sess)
	if err != nil {
		return 0, err
	}
	zn := z.DeviceConfigCustom.ZoneNumber
	if len(opts.Args) > 0 {
		zn, err = strconv.Atoi(opts.Args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid zone number: %v: %w", opts.Args[0], err)
		}
	}
	if zn < 0 || zn > protocol.NumZones {
		return 0, fmt.Errorf("invalid zone number: %v", zn)
	}
	if opts.Writer != nil {
		_, _ = opts.Writer.Write(fmt.Appendf(nil, "zone: %v, status %v", zn, status[zn-1]))
	}
	if z.logger != nil {
		z.logger.Info("zone-status", "zone", zn, "status", status[zn-1])
	}
	return status[zn-1], nil
}

func (z *Zone) Normal(ctx context.Context, opts devices.OperationArgs) (any, bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return nil, false, err
	}
	return nil, status.Logical() == protocol.ZoneNormal, nil
}

func (z *Zone) Violated(ctx context.Context, opts devices.OperationArgs) (any, bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return nil, false, err
	}
	return nil, status.Logical() == protocol.ZoneViolated, nil
}

func (z *Zone) Trouble(ctx context.Context, opts devices.OperationArgs) (any, bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return nil, false, err
	}
	return nil, status.Logical() == protocol.ZoneTrouble, nil
}

func (z *Zone) Bypassed(ctx context.Context, opts devices.OperationArgs) (any, bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return nil, false, err
	}
	return nil, status.Logical() == protocol.ZoneBypassed, nil
}
