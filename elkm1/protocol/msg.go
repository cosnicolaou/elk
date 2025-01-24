// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package protocol

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/cosnicolaou/automation/net/streamconn"
)

var (
	hexLookup = []byte("0123456789ABCDEF")
)

type Request struct{}

func (r Request) RealTime() ([]byte, Response) {
	return formatMessage('r', 'r', nil), Response{Type: 'R', SubType: 'R'}
}

func (r Request) ZoneDefinitions() ([]byte, Response) {
	return formatMessage('z', 'd', nil), Response{Type: 'Z', SubType: 'D'}
}

func (r Request) ZoneName(z int) ([]byte, Response) {
	data := [5]byte{}
	data[0], data[1] = '0', '0'
	data[2] = byte(z/100 + '0')
	data[3] = byte((z%100)/10 + '0')
	data[4] = byte(z%10 + '0')
	return formatMessage('s', 'd', data[:]), Response{Type: 'S', SubType: 'D'}
}

func (r Request) ZoneStatus() ([]byte, Response) {
	return formatMessage('z', 's', nil), Response{Type: 'Z', SubType: 'S'}
}

type Response struct {
	Type, SubType byte
}

func (r Response) Expected(msg []byte) ([]byte, error) {
	t, st, data, err := r.Decode(msg)
	if err != nil {
		return nil, err
	}
	if t != r.Type || st != r.SubType {
		return nil, fmt.Errorf("unexpected message type: got '%c%c', want '%c%c'", t, st, r.Type, r.SubType)
	}
	return data, nil
}

const msgOverhead = 2 + // len
	2 + // type, subtype
	2 + // reserved
	2 + // crc
	2 // crlf

// formatMessage returns a formatted elk1 m1 request message
// included the length and CRC.
func formatMessage(typ, subtype byte, data []byte) []byte {
	// format is:
	// len[2], type[1], subtype[1], data[:], reserved[2]('0'), crc[2], cr, lf

	tl := len(data) + msgOverhead // total length
	buf := make([]byte, tl)
	ml := tl - 4 // (length and crlf) excluded from the in-message length.
	buf[0] = hexLookup[(ml>>4)&0x0f]
	buf[1] = hexLookup[ml&0x0f]
	buf[2] = typ
	buf[3] = subtype
	hd := 4
	copy(buf[hd:], data)
	hd += len(data)
	buf[hd] = '0'
	buf[hd+1] = '0'
	hd += 2
	crc := byte(0)
	for i := 0; i < hd; i++ {
		crc += buf[i]
	}
	crc = (crc ^ 0xff) + 1
	buf[hd] = hexLookup[(crc>>4)&0x0f]
	buf[hd+1] = hexLookup[crc&0x0f]
	buf[hd+2] = '\r'
	buf[hd+3] = '\n'
	return buf
}

func readHexDigit(b byte) uint8 {
	if b >= '0' && b <= '9' {
		return b - '0'
	}
	if b >= 'A' && b <= 'F' {
		return b - 'A' + 10
	}
	if b >= 'a' && b <= 'f' {
		return b - 'a' + 10
	}
	return 0
}

func readHexInt(buf []byte) (int, []byte) {
	return int(readHexDigit(buf[0]))*16 + int(readHexDigit(buf[1])), buf[2:]
}

func readHexIntu8(buf []byte) (uint8, []byte) {
	return readHexDigit(buf[0])*16 + readHexDigit(buf[1]), buf[2:]
}

func (r Response) Decode(buf []byte) (typ, subtype byte, data []byte, err error) {
	if len(buf) < 4 {
		err = fmt.Errorf("message size %v is too short, no size or type bytes", len(buf))
		return
	}
	ml, pbuf := readHexInt(buf) // message length excludes the length and crlf
	typ = pbuf[0]
	subtype = pbuf[1]
	pbuf = pbuf[2:]
	if len(buf) < ml+4 {
		err = fmt.Errorf("message size %v is too short, expected %v", len(buf), ml+4)
		return
	}
	// data excludes the message length, reserved and crc which is included in the message length
	data = slices.Clone(pbuf[:ml-6])
	pbuf = pbuf[len(data):]
	if pbuf[0] != '0' || pbuf[1] != '0' {
		err = fmt.Errorf("invalid reserved bytes: %v", pbuf[:2])
		return
	}
	pbuf = pbuf[2:]                 // skip the reserved bytes
	crc, pbuf := readHexIntu8(pbuf) // read the crc
	for i := 0; i < 4+len(data)+2; i++ {
		crc += buf[i]
	}
	if crc != 0 {
		err = fmt.Errorf("crc error: %v != 0", crc)
	}
	if pbuf[0] != '\r' || pbuf[1] != '\n' {
		err = fmt.Errorf("invalid crlf: %v", pbuf[:2])
	}
	return
}

func readDecInt(buf []byte) (int, []byte) {
	return int(buf[0]-'0')*10 + int(buf[1]-'0'), buf[2:]
}

// ParseTime parses the time from the data returned by an RR reponses or XK message.
func ParseTime(data []byte) (time.Time, bool, error) {
	secs, data := readDecInt(data)
	mins, data := readDecInt(data)
	hours, data := readDecInt(data)
	data = data[1:]               // skip the day of the week
	day, data := readDecInt(data) // day of month
	month, data := readDecInt(data)
	year, data := readDecInt(data)
	year += 2000
	dst := data[0] == '1'
	return time.Date(year, time.Month(month), day, hours, mins, secs, 0, time.Local), dst, nil
}

func (r Response) IsXK(buf []byte) (bool, error) {
	if len(buf) < 4 {
		return false, fmt.Errorf("message size %v is too short, no size or type bytes", len(buf))
	}
	_, pbuf := readHexInt(buf) // message length excludes the length and crlf
	return pbuf[0] == 'X' && pbuf[1] == 'K', nil
}

func (r Response) IsExpected(buf []byte) (bool, error) {
	if len(buf) < 4 {
		return false, fmt.Errorf("message size %v is too short, no size or type bytes", len(buf))
	}
	_, pbuf := readHexInt(buf) // message length excludes the length and crlf
	return pbuf[0] == r.Type && pbuf[1] == r.SubType, nil
}

func rpc(ctx context.Context, sess streamconn.Session, req []byte, resp Response) ([]byte, error) {
	sess.Send(ctx, req)
	var msg []byte
	for {
		msg = sess.ReadUntil(ctx, "\r\n")
		if err := sess.Err(); err != nil {
			return nil, err
		}
		ok, err := resp.IsExpected(msg)
		if err != nil {
			return nil, err
		}
		if ok {
			break
		}
	}
	data, err := resp.Expected(msg)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ParseTextDescription parses the text description of a zone or other entity,
// as in the response to a sd request to obtain the text name of a zone.
func ParseTextDescription(data []byte) (int, string, error) {
	if got, want := len(data), 2+3+16; got != want {
		return 0, "", fmt.Errorf("unexpected response size for text description: got %v, expected %v", got, want)
	}
	data = data[2:]
	id := int(data[0]-'0')*100 + int(data[1]-'0')*10 + int(data[2]-'0')
	data = data[3:]
	return id, string(data), nil
}

// ParseZoneStatus parses the status of a zone as returned by a ZS request.
func ParseZoneStatus(data []byte) (ZoneStatusAll, error) {
	var status ZoneStatusAll
	if got, want := len(data), NumZones; got != want {
		return status, fmt.Errorf("unexpected response size for zone status: got %v, expected %v", got, want)
	}
	for i, s := range data {
		status[i] = ZoneStatus(readHexDigit(s))
	}
	return status, nil
}
