// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
 * License granted by Canonical Limited
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package model_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/CanonicalLtd/serial-vault/datastore"
	"github.com/CanonicalLtd/serial-vault/service"
	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestAPIListHandler(c *check.C) {

	tests := []SuiteTest{
		{false, "GET", "/api/models", nil, 400, "application/json; charset=UTF-8", 0, false, false, 0},
		{false, "GET", "/api/models", nil, 200, "application/json; charset=UTF-8", datastore.SyncUser, true, true, 3},
		{false, "GET", "/api/models", nil, 400, "application/json; charset=UTF-8", datastore.Invalid, false, false, 0},
		{true, "GET", "/api/models", nil, 400, "application/json; charset=UTF-8", 0, true, false, 0},
	}

	for _, t := range tests {
		if t.EnableAuth {
			datastore.Environ.Config.EnableUserAuth = true
		}
		if t.MockError {
			datastore.Environ.DB = &datastore.ErrorMockDB{}
		}

		w := sendAdminAPIRequest(t.Method, t.URL, bytes.NewReader(t.Data), t.Permissions, c)
		c.Assert(w.Code, check.Equals, t.Code)
		c.Assert(w.Header().Get("Content-Type"), check.Equals, t.Type)

		result, err := parseListResponse(w)
		c.Assert(err, check.IsNil)
		c.Assert(result.Success, check.Equals, t.Success)
		c.Assert(len(result.Models), check.Equals, t.List)

		datastore.Environ.Config.EnableUserAuth = false
		if t.MockError {
			datastore.Environ.DB = &datastore.MockDB{}
		}
	}
}

func sendAdminAPIRequest(method, url string, data io.Reader, permissions int, c *check.C) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest(method, url, data)

	switch permissions {
	case datastore.Admin:
		r.Header.Set("user", "sv")
		r.Header.Set("api-key", "ValidAPIKey")
	case datastore.SyncUser:
		r.Header.Set("user", "sync")
		r.Header.Set("api-key", "ValidAPIKey")
	case datastore.Standard:
		r.Header.Set("user", "user1")
		r.Header.Set("api-key", "ValidAPIKey")
	default:
		break
	}

	service.AdminRouter().ServeHTTP(w, r)

	return w
}
