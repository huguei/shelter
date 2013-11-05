package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"labix.org/v2/mgo"
	"log"
	"net"
	"net/mail"
	"shelter/dao"
	"shelter/database/mongodb"
	"shelter/model"
	"time"
)

// This test objective is to verify the domain data persistence. The strategy is to insert
// and search for the information. Check for insert/update consistency (updates don't
// create a new element) and if the object id is set on creation

// List of possible errors in this test. There can be also other errors from low level
// structures
var (
	// Config file path is a mandatory parameter
	ErrConfigFileUndefined = errors.New("Config file path undefined")
)

var (
	// Path for the configuration file with the database connection information
	configFilePath string
)

// DomainDAOTestConfigFile is a structure to store the test configuration file data
type DomainDAOTestConfigFile struct {
	Database struct {
		URI  string
		Name string
	}
}

func init() {
	flag.StringVar(&configFilePath, "config", "", "Configuration file for DomainDAO test")
}

func main() {
	flag.Parse()

	configFile, err := readConfigFile()
	if err == ErrConfigFileUndefined {
		fmt.Println(err.Error())
		fmt.Println("Usage:")
		flag.PrintDefaults()

	} else if err != nil {
		fatalln("Error reading configuration file", err)
	}

	database, err := mongodb.Open(configFile.Database.URI, configFile.Database.Name)
	if err != nil {
		fatalln("Error connecting the database", err)
	}

	// If there was some problem in the last test, there could be some data in the
	// database, so let's clear it to don't affect this test
	if err := database.C(dao.DomainDAOCollection).DropCollection(); err != nil {
		fatalln("Error while trying to clear the scenario in database", err)
	}

	domainLifeCycle(database)
	domainDAOPerformance(database)

	println("SUCCESS!")
}

// Test all phases of the domain life cycle
func domainLifeCycle(database *mgo.Database) {
	domain := newDomain()
	domainDAO := dao.DomainDAO{
		Database: database,
	}

	if err := domainDAO.Save(&domain); err != nil {
		fatalln("Couldn't save domain in database", err)
	}

	if domainRetrieved, err := domainDAO.FindByFQDN(domain.FQDN); err != nil {
		fatalln("Couldn't find domain in database", err)

	} else if !compareDomains(domain, domainRetrieved) {
		fatalln("Domain in being persisted wrongly", nil)
	}

	if err := domainDAO.RemoveByFQDN(domain.FQDN); err != nil {
		fatalln("Error while trying to remove a domain", err)
	}

	if _, err := domainDAO.FindByFQDN(domain.FQDN); err == nil {
		fatalln("Domain was not removed from database", nil)
	}
}

// Check if the DAO operations are optimezed for big volume of data
func domainDAOPerformance(database *mgo.Database) {
	numberOfItems := 10000
	durationTolerance := 5.0 // seconds

	index := mgo.Index{
		Name:       "fqdn",
		Key:        []string{"fqdn"},
		Unique:     true,
		DropDups:   true,
		Background: false,
		Sparse:     false,
	}

	if err := database.C(dao.DomainDAOCollection).EnsureIndex(index); err != nil {
		fatalln("Error trying to set the collection index", err)
	}

	domainDAO := dao.DomainDAO{
		Database: database,
	}

	begin := time.Now()

	for i := 0; i < numberOfItems; i++ {
		domain := model.Domain{
			FQDN: fmt.Sprintf("test%d.com.br", i),
		}

		if err := domainDAO.Save(&domain); err != nil {
			fatalln("Couldn't save domain in database during the performance test", err)
		}
	}

	// Try to find domains from different parts of the whole range to check indexes
	queryRanges := numberOfItems / 4
	fqdn1 := fmt.Sprintf("test%d.com.br", queryRanges)
	fqdn2 := fmt.Sprintf("test%d.com.br", queryRanges*2)
	fqdn3 := fmt.Sprintf("test%d.com.br", queryRanges*3)

	if _, err := domainDAO.FindByFQDN(fqdn1); err != nil {
		fatalln("Couldn't find domain in database during the performance test", err)
	}

	if _, err := domainDAO.FindByFQDN(fqdn2); err != nil {
		fatalln("Couldn't find domain in database during the performance test", err)
	}

	if _, err := domainDAO.FindByFQDN(fqdn3); err != nil {
		fatalln("Couldn't find domain in database during the performance test", err)
	}

	for i := 0; i < numberOfItems; i++ {
		fqdn := fmt.Sprintf("test%d.com.br", i)
		if err := domainDAO.RemoveByFQDN(fqdn); err != nil {
			fatalln("Error while trying to remove a domain during the performance test", err)
		}
	}

	duration := time.Since(begin)
	if duration.Seconds() > durationTolerance {
		fatalln(fmt.Sprintf("Domain DAO operations are too slow (%s)", duration.String()), nil)
	} else {
		println(fmt.Sprintf("Domain DAO operations took %s", time.Since(begin).String()))
	}
}

// Function to read the configuration file
func readConfigFile() (DomainDAOTestConfigFile, error) {
	var configFile DomainDAOTestConfigFile

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

// Function to mock a domain object
func newDomain() model.Domain {
	var domain model.Domain
	domain.FQDN = "rafael.net.br"

	domain.Nameservers = []model.Nameserver{
		{
			Host: "ns1.rafael.net.br",
			IPv4: net.ParseIP("127.0.0.1"),
			IPv6: net.ParseIP("::1"),
		},
		{
			Host: "ns2.rafael.net.br",
			IPv4: net.ParseIP("127.0.0.2"),
		},
	}

	domain.DSSet = []model.DS{
		{
			Keytag:    1234,
			Algorithm: model.DSAlgorithmRSASHA1,
			Digest:    "A790A11EA430A85DA77245F091891F73AA740483",
		},
	}

	owner, _ := mail.ParseAddress("test@rafael.net.br")
	domain.Owners = []*mail.Address{owner}

	return domain
}

// Function to compare if two domains are equal, cannot use operator == because of the
// slices inside the domain object
func compareDomains(d1, d2 model.Domain) bool {
	if d1.Id != d2.Id || d1.FQDN != d2.FQDN {
		return false
	}

	if len(d1.Nameservers) != len(d2.Nameservers) {
		return false
	}

	for i := 0; i < len(d1.Nameservers); i++ {
		// Cannot compare the nameservers directly with operator == because of the
		// pointers for IP addresses
		if d1.Nameservers[i].Host != d2.Nameservers[i].Host ||
			d1.Nameservers[i].IPv4.String() != d2.Nameservers[i].IPv4.String() ||
			d1.Nameservers[i].IPv6.String() != d2.Nameservers[i].IPv6.String() ||
			d1.Nameservers[i].LastStatus != d2.Nameservers[i].LastStatus ||
			d1.Nameservers[i].LastCheckAt != d2.Nameservers[i].LastCheckAt ||
			d1.Nameservers[i].LastOKAt != d2.Nameservers[i].LastOKAt {
			return false
		}
	}

	if len(d1.DSSet) != len(d2.DSSet) {
		return false
	}

	for i := 0; i < len(d1.DSSet); i++ {
		if d1.DSSet[i] != d2.DSSet[i] {
			return false
		}
	}

	if len(d1.Owners) != len(d2.Owners) {
		return false
	}

	for i := 0; i < len(d1.Owners); i++ {
		if d1.Owners[i].String() != d2.Owners[i].String() {
			return false
		}
	}

	return true
}

// Function only to add the test name before the log message. This is useful when you have
// many tests running and logging in the same file, like in a continuous deployment
// scenario
func println(message string) {
	message = fmt.Sprintf("DomainDAO integration test: %s", message)
	log.Println(message)
}

// Function only to add the test name before the log message. This is useful when you have
// many tests running and logging in the same file, like in a continuous deployment
// scenario
func fatalln(message string, err error) {
	message = fmt.Sprintf("DomainDAO integration test: %s", message)
	if err != nil {
		message = fmt.Sprintf("%s. Details: %s", message, err.Error())
	}

	log.Fatalln(message)
}
