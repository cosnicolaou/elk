// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package protocol

import (
	"context"
	"time"

	"github.com/cosnicolaou/automation/net/streamconn"
)

var (
	request Request
)

func GetTime(ctx context.Context, sess streamconn.Session) (time.Time, bool, error) {
	req, resp := request.RealTime()
	data, err := rpc(ctx, sess, req, resp)
	if err != nil {
		return time.Time{}, false, err
	}
	return ParseTime(data)
}
