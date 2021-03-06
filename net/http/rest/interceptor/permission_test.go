// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// interceptor add steps to the REST request before calling the handler
package interceptor

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestACL(t *testing.T) {
	data := []struct {
		ACL           []string
		XForwardedFor string
		XRealIP       string
		RemoteAddr    string
		Code          int
	}{
		{ACL: nil, RemoteAddr: "", Code: 200},
		{ACL: []string{"127.0.0.0/8"}, XForwardedFor: "127.0.0.1", RemoteAddr: "192.168.0.1:1234", Code: 200},
		{ACL: []string{"127.0.0.0/8"}, XForwardedFor: "127.0.0.1,192.168.0.1", RemoteAddr: "192.168.0.1:1234", Code: 200},
		{ACL: []string{"127.0.0.0/8"}, XForwardedFor: "192.168.0.1,127.0.0.1", RemoteAddr: "127.0.0.1:1234", Code: 403},
		{ACL: []string{"127.0.0.0/8"}, XRealIP: "127.0.0.1", RemoteAddr: "192.168.0.1:1234", Code: 200},
		{ACL: []string{"127.0.0.0/8"}, XRealIP: "192.168.0.1", RemoteAddr: "127.0.0.1:1234", Code: 403},
		{ACL: []string{"127.0.0.0/8"}, RemoteAddr: "127.0.0.1:1234", Code: 200},
		{ACL: []string{"127.0.0.0/8"}, RemoteAddr: "127.0.0.1", Code: 500},
		{ACL: []string{"127.0.0.0/8"}, RemoteAddr: "XXX.X.X.X:1234", Code: 500},
		{ACL: []string{"127.0.0.0/8"}, RemoteAddr: "192.168.0.1:1234", Code: 403},
	}

	var permission Permission
	for _, item := range data {
		r, err := http.NewRequest("GET", "/domain/example.com.", nil)
		if err != nil {
			t.Fatal(err)
		}

		ACL = []*net.IPNet{}
		for _, ip := range item.ACL {
			_, ipnet, err := net.ParseCIDR(ip)
			if err != nil {
				t.Fatal(err)
			}

			ACL = append(ACL, ipnet)
		}

		r.RemoteAddr = item.RemoteAddr
		r.Header.Set("X-Forwarded-For", item.XForwardedFor)
		r.Header.Set("X-Real-IP", item.XRealIP)
		w := httptest.NewRecorder()

		permission.Before(w, r)
		permission.After(w, r)

		if w.Code != item.Code {
			t.Errorf("Wrong code defined in permission layer. Expected %d and got %d", item.Code, w.Code)
		}
	}
}
