// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package elkm1

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"cloudeng.io/cmdutil/keystore"
	"gitcom.com/cosnicolaou/elk/elkm1/protocol"
	"github.com/cosnicolaou/automation/devices"
	"github.com/cosnicolaou/automation/net/netutil"
	"github.com/cosnicolaou/automation/net/streamconn"
	"github.com/cosnicolaou/automation/net/streamconn/telnet"
	"github.com/cosnicolaou/automation/net/streamconn/tls"
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
	devices.ControllerConfigCommon
	M1Config `yaml:",inline"`
	logger   *slog.Logger

	mu       sync.Mutex
	ondemand *netutil.OnDemandConnection[streamconn.Session, *M1xep]
}

func NewM1XEP(opts devices.Options) *M1xep {
	m1 := &M1xep{
		logger: opts.Logger.With("protocol", "elk-m1xep"),
	}
	m1.ondemand = netutil.NewOnDemandConnection(m1, streamconn.NewErrorSession)
	return m1
}

func (m1 *M1xep) SetConfig(c devices.ControllerConfigCommon) {
	m1.ControllerConfigCommon = c

}

func (m1 *M1xep) Config() devices.ControllerConfigCommon {
	return m1.ControllerConfigCommon
}

func (m1 *M1xep) CustomConfig() any {
	return m1.M1Config
}

func (m1 *M1xep) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&m1.M1Config); err != nil {
		return err
	}
	if m1.Timeout == 0 {
		return fmt.Errorf("timeout must be specified")
	}
	if m1.KeepAlive == 0 {
		return fmt.Errorf("keep_alive must be specified")
	}
	if m1.TLSVersion != "" {

	}
	m1.ondemand.SetKeepAlive(m1.KeepAlive)
	return nil
}

func (m1 *M1xep) Implementation() any {
	return m1
}

func (m1 *M1xep) Operations() map[string]devices.Operation {
	return map[string]devices.Operation{
		"gettime": func(ctx context.Context, args devices.OperationArgs) error {
			t, dst, err := protocol.GetTime(ctx, m1.Session(ctx))
			dstMsg := "(standard time)"
			if !dst {
				dstMsg = "(daylight saving time)"
			}
			if err == nil {
				fmt.Fprintf(args.Writer, "gettime: %v %v\n", t, dstMsg)
			}
			return err
		},
		"zonenames":  m1.GetZoneNames,
		"zonestatus": m1.GetZoneStatus,
	}
}

func (m1 *M1xep) GetZoneNames(ctx context.Context, args devices.OperationArgs) error {
	defs, err := protocol.GetZoneDefinitions(ctx, m1.Session(ctx))
	if err != nil {
		return err
	}
	names := []string{}
	for i, def := range defs {
		if def == protocol.DisabledZoneType {
			continue
		}
		z := i + 1
		name, err := protocol.GetZoneName(ctx, m1.Session(ctx), z)
		if err != nil {
			return err
		}
		names = append(names, name)
	}
	for i, def := range defs {
		if def == protocol.DisabledZoneType {
			continue
		}
		fmt.Fprintf(args.Writer, "zone %v: %v: %v\n", i+1, def, names[i])
	}
	return nil
}

func (m1 *M1xep) GetZoneStatus(ctx context.Context, args devices.OperationArgs) error {
	status, err := protocol.GetZoneStatusAll(ctx, m1.Session(ctx))
	if err != nil {
		return err
	}
	for i, s := range status {
		if s.Physical() == protocol.ZoneUnconfigured {
			continue
		}
		fmt.Fprintf(args.Writer, "zone %v: %v\n", i+1, s)
	}
	return nil
}
func (m1 *M1xep) OperationsHelp() map[string]string {
	return map[string]string{
		"gettime":    "get the current time from the M1XEP",
		"zonenames":  "get the names of all zones",
		"zonestatus": "get the status of all zones",
	}
}

func (m1 *M1xep) ConnectTLS(ctx context.Context, idle netutil.IdleReset, version string) (streamconn.Session, error) {
	transport, err := tls.Dial(ctx, m1.IPAddress, version, m1.Timeout, m1.logger)
	if err != nil {
		return nil, err
	}
	session := streamconn.NewSession(transport, idle)
	keys := keystore.AuthFromContextForID(ctx, m1.KeyID)
	if err := protocol.M1XEPLogin(ctx, session, keys.User, keys.Token); err != nil {
		session.Close(ctx)
		return nil, err
	}
	return session, nil
}

func (m1 *M1xep) Connect(ctx context.Context, idle netutil.IdleReset) (streamconn.Session, error) {
	if m1.TLSVersion != "" {
		return m1.ConnectTLS(ctx, idle, m1.TLSVersion)
	}
	transport, err := telnet.Dial(ctx, m1.IPAddress, m1.Timeout, m1.logger)
	if err != nil {
		return nil, err
	}
	return streamconn.NewSession(transport, idle), nil
}

func (m1 *M1xep) Disconnect(ctx context.Context, sess streamconn.Session) error {
	return sess.Close(ctx)
}

func (m1 *M1xep) Nil() streamconn.Session {
	return nil
}

// Session returns an authenticated session to the QS processor. If
// an error is encountered then an error session is returned.
func (m1 *M1xep) Session(ctx context.Context) streamconn.Session {
	return m1.ondemand.Connection(ctx)
}

func (m1 *M1xep) Close(ctx context.Context) error {
	return m1.ondemand.Close(ctx)
}
