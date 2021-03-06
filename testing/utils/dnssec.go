// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// utils add features to make the test life easier
package utils

import (
	"github.com/rafaeljusto/shelter/Godeps/_workspace/src/github.com/miekg/dns"
	"github.com/rafaeljusto/shelter/model"
	"time"
)

func GenerateKSKAndSignZone(zone string) (*dns.DNSKEY, *dns.RRSIG, error) {
	return generateKeyAndSignZone(zone, 257)
}

func GenerateZSKAndSignZone(zone string) (*dns.DNSKEY, *dns.RRSIG, error) {
	return generateKeyAndSignZone(zone, 256)
}

func generateKeyAndSignZone(zone string, flags uint16) (*dns.DNSKEY, *dns.RRSIG, error) {
	var globalErr error

	// When creating a lot of keys in a small amount of time, sometimes the systems fails to
	// generate or sign the key. For that reason we try at least 3 times of failure before
	// returning the error. Only this method has this feature because the other ones are not
	// used in performance reports
	for i := 0; i < 3; i++ {
		dnskey := &dns.DNSKEY{
			Hdr: dns.RR_Header{
				Name:   zone,
				Rrtype: dns.TypeDNSKEY,
				Class:  dns.ClassINET,
				Ttl:    86400,
			},
			Flags:     flags,
			Protocol:  3,
			Algorithm: dns.RSASHA1NSEC3SHA1,
		}

		privateKey, err := dnskey.Generate(1024)
		if err != nil {
			globalErr = err
			continue
		}

		rrsig := &dns.RRSIG{
			Hdr: dns.RR_Header{
				Name:   zone,
				Rrtype: dns.TypeRRSIG,
				Class:  dns.ClassINET,
				Ttl:    86400,
			},
			TypeCovered: dns.TypeDNSKEY,
			Algorithm:   dnskey.Algorithm,
			Expiration:  uint32(time.Now().Add(10 * time.Second).Unix()),
			Inception:   uint32(time.Now().Unix()),
			KeyTag:      dnskey.KeyTag(),
			SignerName:  zone,
		}

		if err := rrsig.Sign(privateKey, []dns.RR{dnskey}); err != nil {
			globalErr = err
			continue
		}

		return dnskey, rrsig, nil
	}

	return nil, nil, globalErr
}

// We don't specify a zone for the DNSKEY because we want to reuse the same key in many different
// zones (performance tests)
func GenerateKey() (*dns.DNSKEY, dns.PrivateKey, error) {
	dnskey := &dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   "",
			Rrtype: dns.TypeDNSKEY,
		},
		Flags:     257,
		Protocol:  3,
		Algorithm: dns.RSASHA1NSEC3SHA1,
	}

	privateKey, err := dnskey.Generate(1024)
	return dnskey, privateKey, err
}

func SignKey(zone string, dnskey *dns.DNSKEY, privateKey dns.PrivateKey) (*dns.RRSIG, error) {
	rrsig := &dns.RRSIG{
		Hdr: dns.RR_Header{
			Name:   zone,
			Rrtype: dns.TypeRRSIG,
		},
		TypeCovered: dns.TypeDNSKEY,
		Algorithm:   dnskey.Algorithm,
		Expiration:  uint32(time.Now().Add(10 * time.Second).Unix()),
		Inception:   uint32(time.Now().Unix()),
		KeyTag:      dnskey.KeyTag(),
		SignerName:  zone,
	}

	err := rrsig.Sign(privateKey, []dns.RR{dnskey})
	return rrsig, err
}

func ConvertKeyAlgorithm(algorithm uint8) model.DSAlgorithm {
	switch algorithm {
	case dns.RSAMD5:
		return model.DSAlgorithmRSAMD5
	case dns.DH:
		return model.DSAlgorithmDH
	case dns.DSA:
		return model.DSAlgorithmDSASHA1
	case dns.RSASHA1:
		return model.DSAlgorithmRSASHA1
	case dns.DSANSEC3SHA1:
		return model.DSAlgorithmDSASHA1NSEC3
	case dns.RSASHA1NSEC3SHA1:
		return model.DSAlgorithmRSASHA1NSEC3
	case dns.RSASHA256:
		return model.DSAlgorithmRSASHA256
	case dns.RSASHA512:
		return model.DSAlgorithmRSASHA512
	case dns.ECCGOST:
		return model.DSAlgorithmECCGOST
	case dns.ECDSAP256SHA256:
		return model.DSAlgorithmECDSASHA256
	case dns.ECDSAP384SHA384:
		return model.DSAlgorithmECDSASHA384
	case dns.INDIRECT:
		return model.DSAlgorithmIndirect
	case dns.PRIVATEDNS:
		return model.DSAlgorithmPrivateDNS
	case dns.PRIVATEOID:
		return model.DSAlgorithmPrivateOID
	}

	return model.DSAlgorithmRSASHA1
}
