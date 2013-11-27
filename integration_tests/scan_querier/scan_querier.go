package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"log"
	"net"
	"os"
	"shelter/model"
	"shelter/scan"
	"strconv"
	"strings"
	"time"
)

// List of possible errors in this test. There can be also other errors from low level
// structures
var (
	// Config file path is a mandatory parameter
	ErrConfigFileUndefined = errors.New("Config file path undefined")
	// Input path is mandatory for performance tests
	ErrInputFileUndefined = errors.New("Input file path undefined")
	// Syntax error while parsing the input file. Check the format of the file in
	// readInputFile function comment
	ErrInputFileInvalidFormat = errors.New("Input file has an invalid format")
)

var (
	configFilePath string // Path for the configuration file with all the query parameters
	inputFilePath  string // Path for the input file used for performance tests
)

// Define some scan important variables for the test enviroment, this values indicates the
// size of the channel, the number of concurrently queriers and the UDP max package size
// for firewall problems
const (
	domainsBufferSize = 10
	numberOfQueriers  = 5
	udpMaxSize        = 4096
)

// ScanQuerierTestConfigFile is a structure to store the test configuration file data
type ScanQuerierTestConfigFile struct {
	Server struct {
		Port int
	}
	PerformanceReport struct {
		InputFile string
	}
}

func init() {
	flag.StringVar(&configFilePath, "config", "", "Configuration file for ScanQuerier test")
}

func main() {
	flag.Parse()

	configFile, err := readConfigFile()
	if err == ErrConfigFileUndefined {
		fmt.Println(err.Error())
		fmt.Println("Usage:")
		flag.PrintDefaults()
		return

	} else if err != nil {
		fatalln("Error reading configuration file", err)
	}

	startDNSServer(configFile.Server.Port)
	domainWithNoDNSErrors()
	domainWithNoDNSSECErrors()
	domainDNSTimeout()
	domainDNSUnknownHost()

	println("SUCCESS!")
}

func domainWithNoDNSErrors() {
	domainsToQueryChannel := make(chan *model.Domain, domainsBufferSize)
	domainsToQueryChannel <- &model.Domain{
		FQDN: "br.",
		Nameservers: []model.Nameserver{
			{
				Host: "ns1.br",
				IPv4: net.ParseIP("127.0.0.1"),
			},
		},
	}
	domainsToQueryChannel <- nil // Poison pill

	dns.HandleFunc("br.", func(w dns.ResponseWriter, dnsRequestMessage *dns.Msg) {
		defer w.Close()

		dnsResponseMessage := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id:               dnsRequestMessage.Id,
				Response:         true,
				Opcode:           dns.OpcodeQuery,
				Authoritative:    true,
				RecursionDesired: dnsRequestMessage.RecursionDesired,
				CheckingDisabled: dnsRequestMessage.CheckingDisabled,
				Rcode:            dns.RcodeSuccess,
			},
			Question: dnsRequestMessage.Question,
			Answer: []dns.RR{
				&dns.SOA{
					Hdr: dns.RR_Header{
						Name:   "br.",
						Rrtype: dns.TypeSOA,
					},
					Serial: 2013112600,
				},
			},
		}

		w.WriteMsg(dnsResponseMessage)
	})

	domains := runScan(domainsToQueryChannel)
	for _, domain := range domains {
		if domain.FQDN != "br." ||
			domain.Nameservers[0].LastStatus != model.NameserverStatusOK {
			fatalln("Error checking a well configured DNS domain", nil)
		}
	}

	dns.HandleRemove("br.")
}

func domainWithNoDNSSECErrors() {
	dnskey, rrsig, err := generateKeyAndSignZone("br.")
	if err != nil {
		fatalln("Error creating DNSSEC keys and signatures", err)
	}
	ds := dnskey.ToDS(int(model.DSDigestTypeSHA1))

	domainsToQueryChannel := make(chan *model.Domain, domainsBufferSize)
	domainsToQueryChannel <- &model.Domain{
		FQDN: "br.",
		Nameservers: []model.Nameserver{
			{
				Host: "ns1.br",
				IPv4: net.ParseIP("127.0.0.1"),
			},
		},
		DSSet: []model.DS{
			{
				Keytag:     dnskey.KeyTag(),
				Algorithm:  convertKeyAlgorithm(dnskey.Algorithm),
				DigestType: model.DSDigestTypeSHA1,
				Digest:     ds.Digest,
			},
		},
	}
	domainsToQueryChannel <- nil // Poison pill

	dns.HandleFunc("br.", func(w dns.ResponseWriter, dnsRequestMessage *dns.Msg) {
		defer w.Close()

		dnsResponseMessage := new(dns.Msg)
		defer w.WriteMsg(dnsResponseMessage)

		if dnsRequestMessage.Question[0].Qtype == dns.TypeSOA {
			dnsResponseMessage = &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:               dnsRequestMessage.Id,
					Response:         true,
					Opcode:           dns.OpcodeQuery,
					Authoritative:    true,
					RecursionDesired: dnsRequestMessage.RecursionDesired,
					CheckingDisabled: dnsRequestMessage.CheckingDisabled,
					Rcode:            dns.RcodeSuccess,
				},
				Question: dnsRequestMessage.Question,
				Answer: []dns.RR{
					&dns.SOA{
						Hdr: dns.RR_Header{
							Name:   "br.",
							Rrtype: dns.TypeSOA,
						},
						Serial: 2013112600,
					},
				},
			}

			w.WriteMsg(dnsResponseMessage)

		} else if dnsRequestMessage.Question[0].Qtype == dns.TypeDNSKEY {
			dnsResponseMessage = &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:               dnsRequestMessage.Id,
					Response:         true,
					Opcode:           dns.OpcodeQuery,
					Authoritative:    true,
					RecursionDesired: dnsRequestMessage.RecursionDesired,
					CheckingDisabled: dnsRequestMessage.CheckingDisabled,
					Rcode:            dns.RcodeSuccess,
				},
				Question: dnsRequestMessage.Question,
				Answer: []dns.RR{
					dnskey,
					rrsig,
				},
			}

		}
	})

	domains := runScan(domainsToQueryChannel)
	for _, domain := range domains {
		if domain.FQDN != "br." ||
			domain.DSSet[0].LastStatus != model.DSStatusOK {
			fatalln("Error checking a well configured DNSSEC domain", nil)
		}
	}

	dns.HandleRemove("br.")
}

func domainDNSTimeout() {
	domainsToQueryChannel := make(chan *model.Domain, domainsBufferSize)
	domainsToQueryChannel <- &model.Domain{
		FQDN: "br.",
		Nameservers: []model.Nameserver{
			{
				Host: "google.com.",
			},
		},
	}
	domainsToQueryChannel <- nil // Poison pill

	domains := runScan(domainsToQueryChannel)
	for _, domain := range domains {
		if domain.Nameservers[0].LastStatus != model.NameserverStatusTimeout {
			fatalln("Error checking a timeout domain", nil)
		}
	}
}

func domainDNSUnknownHost() {
	domainsToQueryChannel := make(chan *model.Domain, domainsBufferSize)
	domainsToQueryChannel <- &model.Domain{
		FQDN: "br.",
		Nameservers: []model.Nameserver{
			{
				Host: "br.br.",
			},
		},
	}
	domainsToQueryChannel <- nil // Poison pill

	domains := runScan(domainsToQueryChannel)
	for _, domain := range domains {
		if domain.Nameservers[0].LastStatus != model.NameserverStatusUnknownHost {
			fatalln("Error checking a unknown host", nil)
		}
	}
}

// Method responsable to configure and start scan injector for tests
func runScan(domainsToQueryChannel chan *model.Domain) []*model.Domain {
	var scanQuerierDispacther scan.QuerierDispatcher

	domainsToSaveChannel := scanQuerierDispacther.Start(domainsToQueryChannel, domainsBufferSize,
		numberOfQueriers, udpMaxSize)

	var domains []*model.Domain

	for {
		exit := false

		select {
		case domain := <-domainsToSaveChannel:
			// Detect the poison pills
			if domain == nil {
				exit = true

			} else {
				domains = append(domains, domain)
			}
		}

		if exit {
			break
		}
	}

	return domains
}

func generateKeyAndSignZone(zone string) (*dns.DNSKEY, *dns.RRSIG, error) {
	dnskey := &dns.DNSKEY{
		Hdr: dns.RR_Header{
			Name:   zone,
			Rrtype: dns.TypeDNSKEY,
		},
		Flags:     257,
		Protocol:  3,
		Algorithm: dns.RSASHA1NSEC3SHA1,
	}

	privateKey, err := dnskey.Generate(1024)
	if err != nil {
		return nil, nil, err
	}

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

	if err := rrsig.Sign(privateKey, []dns.RR{dnskey}); err != nil {
		return nil, nil, err
	}

	return dnskey, rrsig, nil
}

func convertKeyAlgorithm(algorithm uint8) model.DSAlgorithm {
	switch algorithm {
	case dns.RSAMD5:
		return model.DSAlgorithmRSAMD5
	case dns.DH:
		return model.DSAlgorithmDH
	case dns.DSA:
		return model.DSAlgorithmDSASHA1
	case dns.ECC:
		return model.DSAlgorithmECC
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

func startDNSServer(port int) {
	// Change the querier DNS port for the scan
	scan.DNSPort = port

	server := dns.Server{
		Net:     "udp",
		Addr:    fmt.Sprintf("localhost:%d", port),
		UDPSize: udpMaxSize,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fatalln("Error starting DNS test server", err)
		}
	}()
}

// Function to read the configuration file
func readConfigFile() (ScanQuerierTestConfigFile, error) {
	var configFile ScanQuerierTestConfigFile

	// Config file path is a mandatory program parameter
	if len(configFilePath) == 0 {
		return configFile, ErrConfigFileUndefined
	}

	confBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return configFile, err
	}

	if err := json.Unmarshal(confBytes, &configFile); err != nil {
		return configFile, err
	}

	return configFile, nil
}

// Function to read input data file for performance tests. The file must use the following
// format:
//
//  <zonename1> <type1> <data1>
//  <zonename2> <type2> <data2>
//  ...
//  <zonenameN> <typeN> <dataN>
//
// Where type can be NS, A, AAAA or DS. All types, except for DS, will have only one field
// in data, for DS we will have four fields. For example:
//
// br.       NS   a.dns.br.
// br.       NS   b.dns.br.
// br.       NS   c.dns.br.
// br.       NS   d.dns.br.
// br.       NS   e.dns.br.
// br.       NS   f.dns.br.
// br.       DS   41674 5 1 EAA0978F38879DB70A53F9FF1ACF21D046A98B5C
// a.dns.br. A    200.160.0.10
// a.dns.br. AAAA 2001:12ff:0:0:0:0:0:10
// b.dns.br. A    200.189.41.10
// c.dns.br. A    200.192.233.10
// d.dns.br. A    200.219.154.10
// d.dns.br. AAAA 2001:12f8:4:0:0:0:0:10
// e.dns.br. A    200.229.248.10
// e.dns.br. AAAA 2001:12f8:1:0:0:0:0:10
// f.dns.br. A    200.219.159.10
//
func readInputFile() ([]*model.Domain, error) {
	// Input file path is necessary when we want to run a performance test, because in this
	// file we have real DNS authoritative servers
	if len(inputFilePath) == 0 {
		return nil, ErrInputFileUndefined
	}

	file, err := os.Open(inputFilePath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	domainsInfo := make(map[string]*model.Domain)
	nameserversInfo := make(map[string]model.Nameserver)

	// Read line by line
	for scanner.Scan() {
		inputParts := strings.Split(scanner.Text(), " ")
		if len(inputParts) < 3 {
			return nil, ErrInputFileInvalidFormat
		}

		zone, rrType := strings.ToLower(inputParts[0]), strings.ToUpper(inputParts[1])

		if rrType == "NS" {
			domain := domainsInfo[zone]
			if domain == nil {
				domain = &model.Domain{
					FQDN: zone,
				}
			}

			domain.Nameservers = append(domain.Nameservers, model.Nameserver{
				Host: strings.ToLower(inputParts[2]),
			})

			domainsInfo[zone] = domain

		} else if rrType == "DS" {
			domain := domainsInfo[zone]
			if domain == nil {
				domain = &model.Domain{
					FQDN: zone,
				}
			}

			if len(inputParts) < 6 {
				return nil, ErrInputFileInvalidFormat
			}

			keytag, err := strconv.Atoi(inputParts[2])
			if err != nil {
				return nil, ErrInputFileInvalidFormat
			}

			algorithm, err := strconv.Atoi(inputParts[3])
			if err != nil {
				return nil, ErrInputFileInvalidFormat
			}

			digestType, err := strconv.Atoi(inputParts[4])
			if err != nil {
				return nil, ErrInputFileInvalidFormat
			}

			domain.DSSet = append(domain.DSSet, model.DS{
				Keytag:     uint16(keytag),
				Algorithm:  model.DSAlgorithm(algorithm),
				DigestType: model.DSDigestType(digestType),
				Digest:     strings.ToUpper(inputParts[5]),
			})

			domainsInfo[zone] = domain

		} else if rrType == "A" {
			nameserver := nameserversInfo[zone]
			nameserver.Host = strings.ToLower(zone)
			nameserver.IPv4 = net.ParseIP(inputParts[2])
			nameserversInfo[zone] = nameserver

		} else if rrType == "AAAA" {
			nameserver := nameserversInfo[zone]
			nameserver.Host = strings.ToLower(zone)
			nameserver.IPv6 = net.ParseIP(inputParts[2])
			nameserversInfo[zone] = nameserver

		} else {
			return nil, ErrInputFileInvalidFormat
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var domains []*model.Domain
	for _, domain := range domainsInfo {
		for index, nameserver := range domain.Nameservers {
			if nameserverGlue, found := nameserversInfo[nameserver.Host]; found {
				domain.Nameservers[index] = nameserverGlue
			}
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// Function only to add the test name before the log message. This is useful when you have
// many tests running and logging in the same file, like in a continuous deployment
// scenario. Prints a simple message without ending the test
func println(message string) {
	message = fmt.Sprintf("ScanQuerier integration test: %s", message)
	log.Println(message)
}

// Function only to add the test name before the log message. This is useful when you have
// many tests running and logging in the same file, like in a continuous deployment
// scenario. Prints an error message and ends the test
func fatalln(message string, err error) {
	message = fmt.Sprintf("ScanQuerier integration test: %s", message)
	if err != nil {
		message = fmt.Sprintf("%s. Details: %s", message, err.Error())
	}

	log.Fatalln(message)
}
