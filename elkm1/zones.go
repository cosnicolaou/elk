// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package elkm1

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"gitcom.com/cosnicolaou/elk/elkm1/protocol"
	"github.com/cosnicolaou/automation/devices"
	"gopkg.in/yaml.v3"
)

type ZoneConfig struct {
	ZoneNumber int `yaml:"zone"`
}

type Zone struct {
	m1DeviceBase
	ZoneConfig
	logger *slog.Logger
}

func (z *Zone) CustomConfig() any {
	return z.ZoneConfig
}

func (z *Zone) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&z.ZoneConfig); err != nil {
		return err
	}
	if z.ZoneNumber > protocol.NumZones {
		return fmt.Errorf("invalid zone number: %v", z.ZoneNumber)
	}
	return nil
}

func (s *Zone) Operations() map[string]devices.Operation {
	return map[string]devices.Operation{}
}

func (z *Zone) OperationsHelp() map[string]string {
	return map[string]string{}
}

func (z *Zone) Conditions() map[string]devices.Condition {
	return map[string]devices.Condition{
		"normal":   z.Normal,
		"violated": z.Violated,
		"trouble":  z.Trouble,
		"bypassed": z.Bypassed,
	}
}

func (z *Zone) ConditionsHelp() map[string]string {
	return map[string]string{}
}

func (z *Zone) logical(ctx context.Context, opts devices.OperationArgs) (protocol.ZoneStatus, error) {
	status, err := protocol.GetZoneStatusAll(ctx, z.m1.Session(ctx))
	if err != nil {
		return 0, err
	}
	zn := z.ZoneNumber
	if len(opts.Args) > 0 {
		zn, err = strconv.Atoi(opts.Args[0])
		if err != nil {
			return 0, fmt.Errorf("invalid zone number: %v: %w", opts.Args[0], err)
		}
	}
	return status[zn-1], nil
}

func (z *Zone) Normal(ctx context.Context, opts devices.OperationArgs) (bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return false, err
	}
	return status.Logical() == protocol.ZoneNormal, nil
}

func (z *Zone) Violated(ctx context.Context, opts devices.OperationArgs) (bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return false, err
	}
	return status.Logical() == protocol.ZoneViolated, nil
}

func (z *Zone) Trouble(ctx context.Context, opts devices.OperationArgs) (bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return false, err
	}
	return status.Logical() == protocol.ZoneTrouble, nil
}

func (z *Zone) Bypassed(ctx context.Context, opts devices.OperationArgs) (bool, error) {
	status, err := z.logical(ctx, opts)
	if err != nil {
		return false, err
	}
	return status.Logical() == protocol.ZoneBypassed, nil
}
