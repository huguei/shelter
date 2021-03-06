// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

// Package dspolicy store the DS record policies for DNSSEC configuration checks
package dspolicy

import (
	"github.com/rafaeljusto/shelter/Godeps/_workspace/src/github.com/miekg/dns"
	"github.com/rafaeljusto/shelter/model"
	"github.com/rafaeljusto/shelter/net/scan/dnsutils"
	"net"
	"strings"
	"time"
)

var (
	// List of all DS policies that are going to be executed in the order defined here. The
	// order is important because the policies depends on each other, assuming that
	// something was already verified
	dsPolicies = []func(*DomainDSPolicy, *dns.Msg) bool{
		(*DomainDSPolicy).dnsHeaderPolicy,
		(*DomainDSPolicy).dnssecPolicy,
	}
)

// DomainDSPolicy store the domain object that is going to be updated during the policies
// executions. The domain object cannot be null
type DomainDSPolicy struct {
	domain *model.Domain // Domain object that stores the last state of the DS records
}

// This function initialize a DomainDSPolicy object, it was created to force the
// programmer to initialize the domain object, so we don't need to check if domain is nil
// inside each method. Maybe there's a better approach (think about)
func NewDomainDSPolicy(domain *model.Domain) DomainDSPolicy {
	return DomainDSPolicy{
		domain: domain,
	}
}

// When there's a error while sending a DS request over the network, this method is
// responsable for detecting any usual problems, something like DNSSEC timeouts. Generic
// kinds of errors should be visible when checking the nameserver policies
func (d *DomainDSPolicy) CheckNetworkError(err error) bool {
	if err == nil {
		return true
	}

	// We can have timeouts only for DNSSEC requests, because usually the response is bigger
	// and firewalls are not configured for big UDP packages, or for DNS over TCP
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		for index, _ := range d.domain.DSSet {
			d.domain.DSSet[index].ChangeStatus(model.DSStatusTimeout)
		}
		return false
	}

	// Other types of network errors are not a specific problem of DNSSEC configuration, so
	// let's just set a status for the user to fix the DNS configuration to make the DNSSEC
	// configuration check possible
	for index, _ := range d.domain.DSSet {
		d.domain.DSSet[index].ChangeStatus(model.DSStatusDNSError)
	}
	return false
}

// Method responsable for running all DS policies. Each nameserver result query can update
// all DS records (because of some error), so this method has a different interface of the
// nameserver policies, it updates the DS records directly in the domain object pointer
// and return true when the DS records are OK or false otherwise
func (d *DomainDSPolicy) Run(dnsResponseMessage *dns.Msg) bool {
	// Something went really wrong, because if we got here there was no network error and it
	// should have a DNS response message, but as a safety check we don't allow to continue
	if dnsResponseMessage == nil {
		return false
	}

	for _, policy := range dsPolicies {
		if !policy(d, dnsResponseMessage) {
			return false
		}
	}

	return true
}

// Policy to check if everything is OK with the DNS package before checking the DNSSEC
// policies, if something is wrong it probably appeared in the nameserver policies results
func (d *DomainDSPolicy) dnsHeaderPolicy(dnsResponseMessage *dns.Msg) bool {
	if dnsResponseMessage.Rcode == dns.RcodeSuccess &&
		dnsResponseMessage.MsgHdr.Authoritative {
		return true
	}

	// Authority errors are not a specific problem of DNSSEC configuration, so let's just
	// set a status for the user to fix the DNS configuration to make the DNSSEC
	// configuration check possible
	for index, _ := range d.domain.DSSet {
		d.domain.DSSet[index].ChangeStatus(model.DSStatusDNSError)
	}
	return false
}

// Verify all DNSSEC rules to see if the DS records of the domains are well configured.
// This method updates directly in the domain object
func (d *DomainDSPolicy) dnssecPolicy(dnsResponseMessage *dns.Msg) bool {
	// Get all DNSSEC public keys
	dnskeys := dnsutils.FilterRRs(dnsResponseMessage.Answer, dns.TypeDNSKEY)

	// Get all signatures from the DNS response
	rrsigs := dnsutils.FilterRRs(dnsResponseMessage.Answer, dns.TypeRRSIG)

	success := true
	for index, ds := range d.domain.DSSet {
		status, signatureExpiration := d.checkDS(ds, dnskeys, rrsigs)
		d.domain.DSSet[index].ChangeStatus(status)
		d.domain.DSSet[index].ExpiresAt = signatureExpiration

		if status != model.DSStatusOK {
			success = false
		}
	}
	return success
}

// For each DS of the domain object we verify a couple of rules with the DNS response
// data. It will return beyond the DS status, the current expiration date retrieved from
// the network, if the expiration date could not be retrieved, we return the current
// expiration date of the DS object
func (d *DomainDSPolicy) checkDS(ds model.DS,
	dnskeys []dns.RR, rrsigs []dns.RR) (model.DSStatus, time.Time) {

	// Find the DNSSEC public key related to the DS
	selectedDNSKEY := d.selectDNSKEY(dnskeys, ds.Keytag)

	if selectedDNSKEY == nil {
		return model.DSStatusNoKey, ds.ExpiresAt
	}

	// Check if the DNSSEC key related to the DS has the security entry point. Check RFCs
	// 3755 and 4034
	if (selectedDNSKEY.Flags & dns.SEP) == 0 {
		return model.DSStatusNoSEP, ds.ExpiresAt
	}

	// Find the signature of the DNSSEC key that signed the keyset
	selectedRRSIG := d.selectRRSIG(rrsigs, ds.Keytag)

	// Keep the same expiration if we don't find a new one
	signatureExpiration := ds.ExpiresAt

	// It's OK to have a DNSKEY without signature, as it is in a key rollover (pre-publish
	// strategy)
	if selectedRRSIG != nil {
		// We store the DS expiration date to alert clients whenever an expiration date is
		// near. There's no status in DS to define a near expiration state, because this
		// isn't a problem
		signatureExpiration = time.Unix(int64(selectedRRSIG.Expiration), 0)

		// Check signature expiration
		if !selectedRRSIG.ValidityPeriod(time.Now()) {
			return model.DSStatusExpiredSignature, signatureExpiration
		}

		// Check signature consistency
		if err := selectedRRSIG.Verify(selectedDNSKEY, dnskeys); err != nil {
			return model.DSStatusSignatureError, signatureExpiration
		}
	}

	// Check DNSKEY hash is the same of the DS digest, hash generated by library is always
	// lower case
	if selectedDNSKEY.ToDS(uint8(ds.DigestType)).Digest != strings.ToLower(ds.Digest) {
		return model.DSStatusNoKey, signatureExpiration
	}

	return model.DSStatusOK, signatureExpiration
}

// selectDNSKEY is responsable for finding the DNSKEY that was used to generate the DS. We
// use the DS keytag to identify the key
func (d *DomainDSPolicy) selectDNSKEY(dnskeys []dns.RR, keytag uint16) *dns.DNSKEY {
	var selectedDNSKEY *dns.DNSKEY
	for _, rr := range dnskeys {
		dnskey, ok := rr.(*dns.DNSKEY)
		if !ok {
			continue
		}

		// The base64 decode method don't deal very well with spaces inside the public key raw
		// data. So we replace it before calculating the KeyTag
		dnskey.PublicKey = strings.Replace(dnskey.PublicKey, " ", "", -1)

		if dnskey.KeyTag() == keytag {
			selectedDNSKEY = dnskey
			break
		}
	}
	return selectedDNSKEY
}

// selectRRSIG will try to find the keyset signature of the DNSKEY related to the DS. We
// use the DS keytag to identify the signatures
func (d *DomainDSPolicy) selectRRSIG(rrsigs []dns.RR, keytag uint16) *dns.RRSIG {
	var selectedRRSIG *dns.RRSIG
	for _, rr := range rrsigs {
		rrsig, ok := rr.(*dns.RRSIG)
		if !ok {
			continue
		}

		// The base64 decode decode don't works well with spaces inside signatures blobs, so
		// we remove them before checking with the DNSKEYs
		rrsig.Signature = strings.Replace(rrsig.Signature, " ", "", -1)

		if rrsig.KeyTag == keytag {
			selectedRRSIG = rrsig
			break
		}
	}
	return selectedRRSIG
}
