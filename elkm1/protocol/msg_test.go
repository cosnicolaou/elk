// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package protocol_test

import (
	"bytes"
	"testing"
	"time"

	"gitcom.com/cosnicolaou/elk/elkm1/protocol"
)

func TestRealtime(t *testing.T) {
	var req protocol.Request
	msg, resp := req.RealTime()
	if got, want := msg, []byte("06rr0056\r\n"); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	data, err := resp.Expected([]byte("16RR0059107251205110006E\r\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rt, dst, err := protocol.ParseTime(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := rt, time.Date(
		2005, 12, 25, 10, 59, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := dst, true; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAsciiDescription(t *testing.T) {
	var req protocol.Request
	msg, resp := req.ZoneName(1)
	if got, want := msg, []byte("0Bsd000010066\r\n"); !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	data, err := resp.Expected([]byte("1BSD01001Front DoorKeypad0089\r\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	id, name, err := protocol.ParseTextDescription(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := id, 1; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := name, "Front DoorKeypad"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
