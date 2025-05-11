// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package protocol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cosnicolaou/automation/net/streamconn"
)

const usernamePrompt = "Username:"
const passwordPrompt = "Password:"
const loginSucessStr = "Elk-M1XEP: Login successful."

var (
	ErrM1XEPLogin = errors.New("Elk-M1XEP login failed")

	loginSucessBytes    = []byte(loginSucessStr)
	usernamePromptBytes = []byte(usernamePrompt)
	passwordPromptBytes = []byte(passwordPrompt)
)

func M1XEPLogin(ctx context.Context, s *streamconn.Session, user, pass string) error {
	if user == "not-set" && pass == "not-set" {
		return nil
	}
	resp, err := s.ReadUntil(ctx, usernamePrompt, "\r\n")
	if err != nil {
		return err
	}
	if !bytes.Contains(resp, usernamePromptBytes) {
		return ErrM1XEPLogin
	}
	s.Send(ctx, []byte(user+"\r\n"))
	resp, err = s.ReadUntil(ctx, passwordPrompt, "\r\n")
	if err != nil {
		return err
	}
	if !bytes.Contains(resp, passwordPromptBytes) {
		return ErrM1XEPLogin
	}
	s.SendSensitive(ctx, []byte(pass+"\r\n"))
	resp, err = s.ReadUntil(ctx, loginSucessStr, usernamePrompt, "\r\n")
	if err := s.Err(); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("user %v: %w", user, ErrM1XEPLogin)
		}
		return err
	}
	if !bytes.Contains(resp, loginSucessBytes) {
		return ErrM1XEPLogin
	}
	return nil
}
