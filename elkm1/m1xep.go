// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package elkm1

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/cmdutil/keystore"
	"github.com/cosnicolaou/automation/devices"
	"github.com/cosnicolaou/automation/net/netutil"
	"github.com/cosnicolaou/automation/net/streamconn"
	"github.com/cosnicolaou/automation/net/streamconn/telnet"
	"github.com/cosnicolaou/automation/net/streamconn/tls"
	"github.com/cosnicolaou/elk/elkm1/protocol"
	"gopkg.in/yaml.v3"
)

type M1Config struct {
	IPAddress  string        `yaml:"ip_address"`
	Timeout    time.Duration `yaml:"timeout"`
	KeepAlive  time.Duration `yaml:"keep_alive"`
	KeyID      string        `yaml:"key_id"`
	TLSVersion string        `yaml:"tls_version"`
	Verbose    bool          `yaml:"verbose"`
}

type M1xep struct {
	devices.ControllerBase[M1Config]
	ondemand *netutil.OnDemandConnection[streamconn.Session, *M1xep]
}

func NewM1XEP(opts devices.Options) *M1xep {
	m1 := &M1xep{}
	m1.ondemand = netutil.NewOnDemandConnection(m1, streamconn.NewErrorSession)
	return m1
}

func (m1 *M1xep) loggingContext(ctx context.Context) context.Context {
	return devices.ContextWithLoggerAttributes(ctx, "protocol", "elk-m1xep")
}

func (m1 *M1xep) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&m1.ControllerConfigCustom); err != nil {
		return err
	}
	if m1.Timeout == 0 {
		return fmt.Errorf("timeout must be specified")
	}
	switch m1.ControllerConfigCustom.TLSVersion {
	case "1.0":
	case "1.2":
	default:
		return fmt.Errorf("unsupported tls version: %v", m1.ControllerConfigCustom.TLSVersion)
	}
	m1.ondemand.SetKeepAlive(m1.ControllerConfigCustom.KeepAlive)
	return nil
}

func (m1 *M1xep) Implementation() any {
	return m1
}

func (m1 *M1xep) Operations() map[string]devices.Operation {
	return map[string]devices.Operation{
		"gettime": func(ctx context.Context, args devices.OperationArgs) (any, error) {
			ctx = m1.loggingContext(ctx)
			t, dst, err := protocol.GetTime(ctx, m1.Session(ctx))
			dstMsg := "(standard time)"
			if !dst {
				dstMsg = "(daylight saving time)"
			}
			if err == nil {
				fmt.Fprintf(args.Writer, "gettime: %v %v\n", t, dstMsg)
			}
			return struct {
				Time string `json:"time"`
			}{Time: t.String()}, err
		},
		"zonenames":  m1.GetZoneNames,
		"zonestatus": m1.GetZoneStatus,
	}
}

type ZoneInfo struct {
	Zone   int    `json:"zone"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

func (m1 *M1xep) GetZoneNames(ctx context.Context, args devices.OperationArgs) (any, error) {
	ctx = m1.loggingContext(ctx)
	defs, err := protocol.GetZoneDefinitions(ctx, m1.Session(ctx))
	if err != nil {
		return nil, err
	}
	names := []string{}
	for i, def := range defs {
		if def == protocol.DisabledZoneType {
			names = append(names, "disabled")
			continue
		}
		z := i + 1
		name, err := protocol.GetZoneName(ctx, m1.Session(ctx), z)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	zi := []ZoneInfo{}
	for i, def := range defs {
		if def == protocol.DisabledZoneType {
			continue
		}
		zi = append(zi, ZoneInfo{Zone: i + 1, Name: names[i]})
		fmt.Fprintf(args.Writer, "zone %v: %v: %v\n", i+1, def, names[i])
	}
	return zi, nil
}

func (m1 *M1xep) GetZoneStatus(ctx context.Context, args devices.OperationArgs) (any, error) {
	ctx = m1.loggingContext(ctx)
	status, err := protocol.GetZoneStatusAll(ctx, m1.Session(ctx))
	if err != nil {
		return nil, err
	}
	zi := []ZoneInfo{}
	for i, s := range status {
		if s.Physical() == protocol.ZoneUnconfigured {
			continue
		}
		zi = append(zi, ZoneInfo{Zone: i + 1, Status: s.String()})
		fmt.Fprintf(args.Writer, "zone %v: %v\n", i+1, s)
	}
	return zi, nil
}

func (m1 *M1xep) OperationsHelp() map[string]string {
	return map[string]string{
		"gettime":    "get the current time from the M1XEP",
		"zonenames":  "get the names of all zones",
		"zonestatus": "get the status of all zones",
	}
}

func (m1 *M1xep) ConnectTLS(ctx context.Context, idle netutil.IdleReset, version string) (streamconn.Session, error) {
	ctx = m1.loggingContext(ctx)
	transport, err := tls.Dial(ctx, m1.ControllerConfigCustom.IPAddress, version, m1.Timeout)
	if err != nil {
		return nil, err
	}
	session := streamconn.NewSession(transport, idle)
	if m1.ControllerConfigCustom.KeyID == "not-set" {
		return session, nil
	}
	keys := keystore.AuthFromContextForID(ctx, m1.ControllerConfigCustom.KeyID)
	if err := protocol.M1XEPLogin(ctx, session, keys.User, keys.Token); err != nil {
		session.Close(ctx)
		return nil, err
	}
	return session, nil
}

func (m1 *M1xep) Connect(ctx context.Context, idle netutil.IdleReset) (streamconn.Session, error) {
	if m1.ControllerConfigCustom.TLSVersion != "" {
		return m1.ConnectTLS(ctx, idle, m1.ControllerConfigCustom.TLSVersion)
	}
	transport, err := telnet.Dial(ctx, m1.ControllerConfigCustom.IPAddress, m1.Timeout)
	if err != nil {
		return nil, err
	}
	return streamconn.NewSession(transport, idle), nil
}

func (m1 *M1xep) Disconnect(ctx context.Context, sess streamconn.Session) error {
	return sess.Close(ctx)
}

// Session returns an authenticated session to the QS processor. If
// an error is encountered then an error session is returned.
func (m1 *M1xep) Session(ctx context.Context) streamconn.Session {
	return m1.ondemand.Connection(ctx)
}

func (m1 *M1xep) Close(ctx context.Context) error {
	return m1.ondemand.Close(ctx)
}
