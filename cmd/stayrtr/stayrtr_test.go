package main

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	rtr "github.com/bgp/stayrtr/lib"
	"github.com/bgp/stayrtr/prefixfile"
	"github.com/google/go-cmp/cmp"
)

func TestProcessData(t *testing.T) {
	var stuff []prefixfile.VRPJson
	NowUnix := int(time.Now().Unix())
	ExpiredTime := 1337

	stuff = append(stuff,
		prefixfile.VRPJson{
			Prefix:  "192.168.0.0/24",
			Length:  24,
			ASN:     123,
			TA:      "testrir",
			Expires: &NowUnix,
		},
		prefixfile.VRPJson{
			Prefix: "192.168.0.0/24",
			Length: 24,
			TA:     "testrir",
		},
		prefixfile.VRPJson{
			Prefix: "2001:db8::/32",
			Length: 33,
			ASN:    "AS123",
			TA:     "testrir",
		},
		prefixfile.VRPJson{
			Prefix: "192.168.1.0/24",
			Length: 25,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. Length is 0
		prefixfile.VRPJson{
			Prefix: "192.168.1.0/24",
			Length: 0,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. Length less than prefix length
		prefixfile.VRPJson{
			Prefix: "192.168.1.0/24",
			Length: 16,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. 129 is invalid for IPv6
		prefixfile.VRPJson{
			Prefix: "2001:db8::/32",
			Length: 129,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. 33 is invalid for IPv4
		prefixfile.VRPJson{
			Prefix: "192.168.1.0/24",
			Length: 33,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. Not a prefix
		prefixfile.VRPJson{
			Prefix: "192.168.1.0",
			Length: 24,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. Not a prefix
		prefixfile.VRPJson{
			Prefix: "👻",
			Length: 24,
			ASN:    123,
			TA:     "testrir",
		},
		// Invalid. Invalid ASN string
		prefixfile.VRPJson{
			Prefix: "192.168.1.0/22",
			Length: 22,
			ASN:    "ASN123",
			TA:     "testrir",
		},
		// Invalid. Has expired
		prefixfile.VRPJson{
			Prefix:  "192.168.2.0/24",
			Length:  24,
			ASN:     124,
			TA:      "testrir",
			Expires: &ExpiredTime,
		},
	)
	got, _, _, count, v4count, v6count := processData(stuff, nil, nil)
	want := []rtr.VRP{
		{
			Prefix: mustParseIPNet("2001:db8::/32"),
			MaxLen: 33,
			ASN:    123,
		},
		{
			Prefix: mustParseIPNet("192.168.1.0/24"),
			MaxLen: 25,
			ASN:    123,
		},
		{
			Prefix: mustParseIPNet("192.168.0.0/24"),
			MaxLen: 24,
			ASN:    123,
		},
	}
	if count != 3 || v4count != 2 || v6count != 1 {
		t.Errorf("Wanted count = 3, v4count = 2, v6count = 1, but got %d, %d, %d", count, v4count, v6count)
	}

	if !cmp.Equal(got, want) {
		t.Errorf("Want (%+v), Got (%+v)", want, got)
	}
}

// mustParseIPNet is a test helper function to return a net.IPNet
// This should only be called in test code, and it'll panic on test set up
// if unable to parse.
func mustParseIPNet(prefix string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(prefix)
	if err != nil {
		panic(err)
	}
	return *ipnet
}

func BenchmarkDecodeJSON(b *testing.B) {
	json, err := os.ReadFile("test.rpki.json")
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		decodeJSON(json)
	}
}

func TestJson(t *testing.T) {
	json, err := os.ReadFile("smalltest.rpki.json")
	if err != nil {
		panic(err)
	}
	got, err := decodeJSON(json)
	if err != nil {
		t.Errorf("Unable to decode json: %v", err)
	}

	Ex1 := 1627568318
	Ex2 := 1627575699

	want := (&prefixfile.VRPList{
		Metadata: prefixfile.MetaData{
			Counts:    2,
			Buildtime: "2021-07-27T18:56:02Z",
		},
		Data: []prefixfile.VRPJson{
			{Prefix: "1.0.0.0/24",
				Length:  24,
				ASN:     float64(13335),
				TA:      "apnic",
				Expires: &Ex1,
			},
			{
				Prefix:  "2001:200:136::/48",
				Length:  48,
				ASN:     "AS9367",
				TA:      "apnic",
				Expires: &Ex2,
			},
		},
	})

	if !cmp.Equal(got, want) {
		t.Errorf("Got (%v), Wanted (%v)", got, want)
	}

}

func TestNewSHA256(t *testing.T) {
	want := "8eddd6897b244bb4d045ff811128b50b53ed85d19a9d1b756a0a400e82b23c2f"
	got := fmt.Sprintf("%x", newSHA256([]byte("☘️")))
	if got != want {
		t.Errorf("Got (%s), Wanted (%s)", got, want)
	}
}
