// Copyright 2014 Rafael Dantas Justo. All rights reserved.
// Use of this source code is governed by a GPL
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/nsf/termbox-go"
	"github.com/rafaeljusto/shelter/config"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	configFilePath       = "/usr/shelter/etc/shelter.conf"
	sampleConfigFilePath = "/usr/shelter/etc/shelter.conf.sample"
)

var (
	scanModule         = false
	restModule         = false
	webClientModule    = false
	notificationModule = false
)

var (
	hostnameOrIPInput = regexp.MustCompile("[0-9A-Za-z\\-\\.\\:]")
	hostnameInput     = regexp.MustCompile("[0-9A-Za-z\\-\\.]")
	alphaNumericInput = regexp.MustCompile("[0-9A-Za-z]")
	numericInput      = regexp.MustCompile("[0-9]")
	ipRangeInput      = regexp.MustCompile("[0-9a-fA-F\\:\\./]")
	secretAlphabet    = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()-_=+,.?/:;{}[]`~")
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
		}
	}()

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// Initialize configuration file

		if err := config.LoadConfig(sampleConfigFilePath); err != nil {
			log.Fatalln(err)
			return
		}

		if !readEnabledModules() ||
			!readDatabaseParameters() ||
			!readRESTParameters() {

			return
		}

		jsonConfig, err := json.MarshalIndent(config.ShelterConfig, " ", " ")
		if err != nil {
			log.Fatalln(err)
			return
		}

		if err := ioutil.WriteFile(configFilePath, jsonConfig, 0664); err != nil {
			log.Fatalln(err)
			return
		}

	} else {
		// Update current configuration file
	}
}

func readEnabledModules() bool {
	options := []string{
		"Scan",
		"REST",
		"Web Client",
		"Notification",
	}

	title := "Modules"
	description := "Please select the modules that are going to be enabled:"

	selectedOptions, continueProcessing :=
		manageInputOptionsScreen(title, description, options, nil)

	if !continueProcessing {
		return false
	}

	for _, option := range selectedOptions {
		switch option {
		case 0:
			scanModule = true
		case 1:
			restModule = true
		case 2:
			webClientModule = true
		case 3:
			notificationModule = true
		}
	}

	config.ShelterConfig.Scan.Enabled = scanModule
	config.ShelterConfig.RESTServer.Enabled = restModule
	config.ShelterConfig.WebClient.Enabled = webClientModule
	config.ShelterConfig.Notification.Enabled = notificationModule

	return true
}

func readDatabaseParameters() bool {
	var address, name string
	var port int
	var continueProcessing bool

	if restModule || notificationModule || scanModule {
		address, continueProcessing = readDatabaseHost()
		if !continueProcessing {
			return false
		}

		port, continueProcessing = readDatabasePort()
		if !continueProcessing {
			return false
		}

		name, continueProcessing = readDatabaseName()
		if !continueProcessing {
			return false
		}
	}

	config.ShelterConfig.Database.URI = fmt.Sprintf("%s:%d", address, port)
	config.ShelterConfig.Database.Name = name
	return true
}

func readDatabaseHost() (string, bool) {
	host := "localhost_________________________________________"
	title := "Database Configurations"
	description := "Please inform the host (IP or domain) of MongoDB database:"
	return manageInputTextScreen(title, description, host, hostnameOrIPInput,
		func(input string) (bool, string) {
			if len(input) == 0 {
				return false, "Host cannot be empty"
			}

			return true, ""
		})
}

func readDatabasePort() (int, bool) {
	port := "27017"
	title := "Database Configurations"
	description := "Please inform the port for the MongoDB database:"
	return manageInputNumberScreen(title, description, port)
}

func readDatabaseName() (string, bool) {
	name := "shelter___________________________________________"
	title := "Database Configurations"
	description := "Please inform the name of the MongoDB database:"
	return manageInputTextScreen(title, description, name, alphaNumericInput,
		func(input string) (bool, string) {
			if len(input) == 0 {
				return false, "Database name cannot be empty"
			}

			return true, ""
		})
}

func readRESTParameters() bool {
	return readRESTListeners() && readRESTACL() && readRESTSecret()
}

func readRESTListeners() bool {
	var addresses []string
	var port int
	var useTLS, generateCerts bool
	var continueProcessing bool

	if restModule || webClientModule {
		addresses, continueProcessing = readRESTAddresses()
		if !continueProcessing {
			return false
		}

		port, continueProcessing = readRESTPort()
		if !continueProcessing {
			return false
		}
	}

	if restModule {
		useTLS, generateCerts, continueProcessing = readRESTTLS()
		if !continueProcessing {
			return false
		}

		if generateCerts {
			hostname, continueProcessing := readRESTCertsParams()
			if !continueProcessing {
				return false
			}

			generateCertificates(hostname)
		}
	}

	config.ShelterConfig.RESTServer.Listeners = []struct {
		IP   string
		Port int
		TLS  bool
	}{}

	for _, address := range addresses {
		config.ShelterConfig.RESTServer.Listeners = append(config.ShelterConfig.RESTServer.Listeners, struct {
			IP   string
			Port int
			TLS  bool
		}{
			IP:   address,
			Port: port,
			TLS:  useTLS,
		})
	}

	return true
}

func generateCertificates(hostname string) {
	cmd := exec.Command("/usr/shelter/bin/generate_cert", "--host", hostname)
	if err := cmd.Run(); err != nil {
		log.Println("Error generating certificates. Details:", err)
		return
	}

	err := os.MkdirAll("/usr/shelter/etc/keys", os.ModeDir|0600)
	if err != nil {
		log.Println("Error creating certificates directory. Details:", err)
		return
	}

	if err := moveFile("/usr/shelter/etc/keys/cert.pem", "cert.pem"); err != nil {
		log.Println("Error copying file cert.pem. Details:", err)
	}

	if err := moveFile("/usr/shelter/etc/keys/key.pem", "key.pem"); err != nil {
		log.Println("Error copying file key.pem. Details:", err)
	}
}

func readRESTAddresses() ([]string, bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return nil, true
	}

	var options []string
	for _, i := range interfaces {
		addrs, err := i.Addrs()
		if err != nil {
			log.Println(err)
			return nil, true
		}

		for _, a := range addrs {
			ip := a.String()
			ip = ip[:strings.Index(ip, "/")]
			options = append(options, fmt.Sprintf("%s", ip))
		}
	}

	title := "REST Configurations"
	description := "Please select the IP addresses that you want to listen:"

	selectedOptions, continueProcessing :=
		manageInputOptionsScreen(title, description, options, nil)

	if !continueProcessing {
		return nil, false
	}

	var selectedAddresses []string
	for _, option := range selectedOptions {
		selectedAddresses = append(selectedAddresses, options[option])
	}
	return selectedAddresses, true
}

func readRESTPort() (int, bool) {
	port := "4443_"
	title := "REST Configurations"
	description := "Please inform the port that you want to listen:"
	return manageInputNumberScreen(title, description, port)
}

func readRESTTLS() (useTLS, generateCerts, continueProcessing bool) {
	options := []string{
		"Use TLS on interfaces (HTTPS)",
		"Generate self-signed certificates automatically (valid for 1 year)",
	}

	title := "REST Configurations"
	description := "Please select the following TLS options:"

	selectedOptions, continueProcessing :=
		manageInputOptionsScreen(title, description, options, func(selectedOptions []int, isNew bool) []int {
			// Automatically certificates generation cannot exist without TLS
			if len(selectedOptions) == 1 && selectedOptions[0] == 1 {
				if isNew {
					selectedOptions = append(selectedOptions, 0)
				} else {
					selectedOptions = []int{}
				}
			}

			return selectedOptions
		})

	if !continueProcessing {
		return
	}

	for _, option := range selectedOptions {
		if option == 0 {
			useTLS = true
		} else if option == 1 {
			generateCerts = true
		}
	}

	return
}

func readRESTCertsParams() (string, bool) {
	host := "localhost_________________________________________"
	title := "REST Configurations"
	description := "Please inform the hostname of the certificate:"
	return manageInputTextScreen(title, description, host, hostnameInput,
		func(input string) (bool, string) {
			if len(input) == 0 {
				return false, "Certificate hostname cannot be empty"
			}

			return true, ""
		})
}

func readRESTACL() bool {
	acl := "127.0.0.1/8_______________________________________"
	title := "REST Configurations"
	description := "Please inform IP ranges that will have access to " +
		"the REST server (separeted by comma):"

	acl, continueProcessing :=
		manageInputTextScreen(title, description, acl, ipRangeInput,
			func(input string) (bool, string) {
				if len(input) == 0 {
					return false, "ACL cannot be empty"
				}

				aclParts := strings.Split(input, ",")
				for _, aclPart := range aclParts {
					aclPart = strings.TrimSpace(aclPart)
					if _, _, err := net.ParseCIDR(aclPart); err != nil {
						return false, "IP range " + aclPart + " is invalid"
					}
				}

				return true, ""
			})

	if !continueProcessing {
		return false
	}

	config.ShelterConfig.RESTServer.ACL = []string{}
	aclParts := strings.Split(acl, ",")
	for _, aclPart := range aclParts {
		aclPart = strings.TrimSpace(aclPart)
		config.ShelterConfig.RESTServer.ACL =
			append(config.ShelterConfig.RESTServer.ACL, aclPart)
	}

	return true
}

func readRESTSecret() bool {
	keyId, generateAutomatically, continueProcessing := readRESTSecretId()

	if !continueProcessing {
		return false
	}

	var secret string
	if generateAutomatically {
		for i := 0; i < 30; i++ {
			secret += string(secretAlphabet[rand.Int()%len(secretAlphabet)])
		}

	} else {
		secret, continueProcessing = readRESTSecretContent(keyId)
		if !continueProcessing {
			return false
		}
	}

	config.ShelterConfig.RESTServer.Secrets[keyId] = secret
	return true
}

func readRESTSecretId() (keyId string, generateSecret bool, continueProcessing bool) {
	keyId = "key01_______________"
	options := []string{
		"Generate shared secret automatically",
	}

	title := "REST Configurations"
	description := "Please inform the shared secret identification:"

	keyId, selectedOptions, continueProcessing :=
		manageInputTextOptionsScreen(title, description, keyId, alphaNumericInput,
			func(input string) (bool, string) {
				if len(input) == 0 {
					return false, "Certificate hostname cannot be empty"
				}

				return true, ""
			}, options, nil)

	return keyId, len(selectedOptions) > 0, continueProcessing
}

func readRESTSecretContent(keyId string) (string, bool) {
	secret := "__________________________________________________"
	title := "REST Configurations"
	description := "Please inform the shared secret for " + keyId + ":"

	return manageInputTextScreen(title, description, secret, alphaNumericInput,
		func(input string) (bool, string) {
			if len(input) == 0 {
				return false, "Certificate hostname cannot be empty"
			}

			return true, ""
		})
}

func readInput(inputsDraw func(), inputsAction func(termbox.Event) bool) bool {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()

	termbox.SetInputMode(termbox.InputEsc)
	draw(inputsDraw)

	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			if ev.Key == termbox.KeyEsc {
				return false
			}

			if !inputsAction(ev) {
				return true
			}

		case termbox.EventResize:
			draw(inputsDraw)
		}
	}
}

func draw(inputsDraw func()) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	windowWidth, windowsHeight := termbox.Size()

	// Background
	for i := 0; i < windowWidth; i++ {
		for j := 0; j < windowsHeight; j++ {
			termbox.SetCell(i, j, 0x0, termbox.ColorWhite, termbox.ColorBlue)
		}
	}

	// Edges
	termbox.SetCell(0, 0, 0x250C, termbox.ColorWhite, termbox.ColorBlue)
	termbox.SetCell(windowWidth-1, 0, 0x2510, termbox.ColorWhite, termbox.ColorBlue)
	termbox.SetCell(0, windowsHeight-1, 0x2514, termbox.ColorWhite, termbox.ColorBlue)
	termbox.SetCell(windowWidth-1, windowsHeight-1, 0x2518, termbox.ColorWhite, termbox.ColorBlue)

	// Logo
	termbox.SetCell((windowWidth/2)-7, 2, 0xff33, termbox.ColorWhite, termbox.ColorBlue) // S
	termbox.SetCell((windowWidth/2)-5, 2, 0xff28, termbox.ColorWhite, termbox.ColorBlue) // H
	termbox.SetCell((windowWidth/2)-3, 2, 0xff25, termbox.ColorWhite, termbox.ColorBlue) // E
	termbox.SetCell((windowWidth/2)-1, 2, 0xff2c, termbox.ColorWhite, termbox.ColorBlue) // L
	termbox.SetCell((windowWidth/2)+1, 2, 0xff34, termbox.ColorWhite, termbox.ColorBlue) // T
	termbox.SetCell((windowWidth/2)+3, 2, 0xff25, termbox.ColorWhite, termbox.ColorBlue) // E
	termbox.SetCell((windowWidth/2)+5, 2, 0xff32, termbox.ColorWhite, termbox.ColorBlue) // R

	// Footer tip
	writeText("Press ESC to quit", windowWidth-20, windowsHeight-2)

	inputsDraw()

	termbox.Flush()
}

func writeTitle(text string, x, y int) {
	xTmp := x
	for _, character := range text {
		termbox.SetCell(xTmp, y, rune(character), termbox.ColorWhite, termbox.ColorBlue)
		xTmp += 1
	}

	xTmp = x
	for xTmp <= len(text)+1 {
		termbox.SetCell(xTmp, y+1, 0x2015, termbox.ColorWhite, termbox.ColorBlue)
		xTmp += 1
	}
}

func writeText(text string, x, y int) {
	for _, character := range text {
		termbox.SetCell(x, y, rune(character), termbox.ColorWhite, termbox.ColorBlue)
		x += 1
	}
}

func writeOptions(options []string, x, y int) {
	for index, option := range options {
		writeText("[ ] "+option, x, y+index)
	}
}

func manageInputOptionsScreen(
	title, description string,
	options []string,
	checkConsistency func([]int, bool) []int,
) ([]int, bool) {

	overOption := -1
	var selectedOptions []int

	// By default all options are selected
	for index, _ := range options {
		selectedOptions = append(selectedOptions, index)
	}

	inputsDraw := func() {
		writeTitle(title, 2, 4)
		writeText(description, 2, 7)
		writeOptions(options, 2, 9)

		_, windowsHeight := termbox.Size()
		writeText("[TAB] Move over options", 2, windowsHeight-4)
		writeText("[SPACE] Select an option", 2, windowsHeight-3)
		writeText("[ENTER] Continue", 2, windowsHeight-2)

		for _, selectedOption := range selectedOptions {
			termbox.SetCell(3, 9+selectedOption, 0x221a, termbox.ColorYellow, termbox.ColorBlue)
		}

		if overOption > -1 {
			termbox.SetCell(2, 9+overOption, rune('['), termbox.ColorYellow, termbox.ColorBlue)
			termbox.SetCell(4, 9+overOption, rune(']'), termbox.ColorYellow, termbox.ColorBlue)
		}
	}

	restInputsAction := func(ev termbox.Event) bool {
		switch ev.Key {
		case termbox.KeyTab:
			// Move to the next option

			overOption += 1
			if overOption >= len(options) {
				overOption = 0
			}

		case termbox.KeySpace:
			// Select the option

			found := false
			for index, selectedOption := range selectedOptions {
				if selectedOption == overOption {
					if len(selectedOptions) == 1 {
						selectedOptions = []int{}
					} else if index == 0 {
						selectedOptions = selectedOptions[index+1:]
					} else if index == len(selectedOptions)-1 {
						selectedOptions = selectedOptions[:index]
					} else {
						selectedOptions = append(selectedOptions[:index], selectedOptions[index+1:]...)
					}
					found = true
				}
			}

			if !found {
				selectedOptions = append(selectedOptions, overOption)
			}

			if checkConsistency != nil {
				selectedOptions = checkConsistency(selectedOptions, !found)
			}

		case termbox.KeyEnter:
			// Finish reading inputs
			return false
		}

		draw(inputsDraw)
		return true
	}

	if !readInput(inputsDraw, restInputsAction) {
		return nil, false
	}

	return selectedOptions, true
}

func manageInputTextScreen(
	title, description, input string,
	allowedInput *regexp.Regexp,
	validate func(string) (bool, string),
) (string, bool) {

	inputPosition := 0

	inputsDraw := func() {
		writeTitle(title, 2, 4)
		writeText(description, 2, 7)
		writeText(input, 2, 9)

		if inputPosition < len(input) {
			termbox.SetCell(2+inputPosition, 9, rune(input[inputPosition]), termbox.ColorWhite, termbox.ColorYellow)
		}

		_, windowsHeight := termbox.Size()
		writeText("[ENTER] Continue", 2, windowsHeight-2)
	}

	restInputsAction := func(ev termbox.Event) bool {
		switch ev.Key {
		case termbox.KeyBackspace, termbox.KeyBackspace2:
			inputPosition -= 1
			if inputPosition < 0 {
				inputPosition = 0
			}

			input = input[:inputPosition] + "_" + input[inputPosition+1:]

		case termbox.KeyDelete:
			if inputPosition < len(input) {
				input = input[:inputPosition] + input[inputPosition+1:] + "_"
			}

		case termbox.KeyEnter:
			if validate != nil {
				inputTmp := strings.Replace(input, "_", "", -1)
				valid, msg := validate(inputTmp)

				if !valid {
					draw(func() {
						inputsDraw()
						writeText("ERROR: "+msg, 2, 11)
					})

					return true
				}
			}

			// Finish reading inputs
			return false

		default:
			if allowedInput.MatchString(string(ev.Ch)) &&
				inputPosition < len(input) {

				input = input[:inputPosition] + string(ev.Ch) + input[inputPosition+1:]

				inputPosition += 1
				if inputPosition > len(input) {
					inputPosition = len(input)
				}
			}
		}

		draw(inputsDraw)
		return true
	}

	if !readInput(inputsDraw, restInputsAction) {
		return "", false
	}

	return strings.Replace(input, "_", "", -1), true
}

func manageInputNumberScreen(title, description string, number string) (int, bool) {
	number, continueProcessing :=
		manageInputTextScreen(title, description, number, numericInput,
			func(input string) (bool, string) {
				if len(input) == 0 {
					return false, "You must inform a number"
				}

				return true, ""
			})

	if !continueProcessing {
		return 0, false
	}

	numberConverted, _ := strconv.Atoi(number)
	return numberConverted, true
}

func manageInputTextOptionsScreen(
	title, description string,
	input string,
	allowedInput *regexp.Regexp,
	validate func(string) (bool, string),
	options []string,
	checkConsistency func([]int, bool) []int,
) (string, []int, bool) {

	inputPosition := 0
	overOption := 0
	var selectedOptions []int

	// By default all options are selected except for the first one (that is the input text)
	for index, _ := range options {
		selectedOptions = append(selectedOptions, index)
	}

	inputsDraw := func() {
		writeTitle(title, 2, 4)
		writeText(description, 2, 7)
		writeText(input, 2, 9)
		writeOptions(options, 2, 11)

		_, windowsHeight := termbox.Size()
		writeText("[TAB] Move over options", 2, windowsHeight-4)
		writeText("[SPACE] Select an option", 2, windowsHeight-3)
		writeText("[ENTER] Continue", 2, windowsHeight-2)

		for _, selectedOption := range selectedOptions {
			termbox.SetCell(3, 11+selectedOption, 0x221a, termbox.ColorYellow, termbox.ColorBlue)
		}

		if inputPosition < len(input) && overOption == 0 {
			termbox.SetCell(2+inputPosition, 9, rune(input[inputPosition]), termbox.ColorWhite, termbox.ColorYellow)
		}

		if overOption > 0 {
			termbox.SetCell(2, 10+overOption, rune('['), termbox.ColorYellow, termbox.ColorBlue)
			termbox.SetCell(4, 10+overOption, rune(']'), termbox.ColorYellow, termbox.ColorBlue)
		}
	}

	restInputsAction := func(ev termbox.Event) bool {
		switch ev.Key {
		case termbox.KeyTab:
			// Move to the next option

			overOption += 1
			if overOption >= len(options)+1 {
				overOption = 0
			}

		case termbox.KeyBackspace, termbox.KeyBackspace2:
			if overOption != 0 {
				break
			}

			inputPosition -= 1
			if inputPosition < 0 {
				inputPosition = 0
			}

			input = input[:inputPosition] + "_" + input[inputPosition+1:]

		case termbox.KeyDelete:
			if overOption != 0 {
				break
			}

			if inputPosition < len(input) {
				input = input[:inputPosition] + input[inputPosition+1:] + "_"
			}

		case termbox.KeySpace:
			if overOption == 0 {
				break
			}

			// Select the option

			found := false
			for index, selectedOption := range selectedOptions {
				if selectedOption == overOption-1 {
					if len(selectedOptions) == 1 {
						selectedOptions = []int{}
					} else if index == 0 {
						selectedOptions = selectedOptions[index+1:]
					} else if index == len(selectedOptions)-1 {
						selectedOptions = selectedOptions[:index]
					} else {
						selectedOptions = append(selectedOptions[:index], selectedOptions[index+1:]...)
					}
					found = true
				}
			}

			if !found {
				selectedOptions = append(selectedOptions, overOption-1)
			}

			if checkConsistency != nil {
				selectedOptions = checkConsistency(selectedOptions, !found)
			}

		case termbox.KeyEnter:
			if validate != nil {
				inputTmp := strings.Replace(input, "_", "", -1)
				valid, msg := validate(inputTmp)

				if !valid {
					draw(func() {
						inputsDraw()
						writeText("ERROR: "+msg, 2, 13)
					})

					return true
				}
			}

			// Finish reading inputs
			return false

		default:
			if overOption != 0 {
				break
			}

			if allowedInput.MatchString(string(ev.Ch)) &&
				inputPosition < len(input) {

				input = input[:inputPosition] + string(ev.Ch) + input[inputPosition+1:]

				inputPosition += 1
				if inputPosition > len(input) {
					inputPosition = len(input)
				}
			}
		}

		draw(inputsDraw)
		return true
	}

	if !readInput(inputsDraw, restInputsAction) {
		return "", nil, false
	}

	return strings.Replace(input, "_", "", -1), selectedOptions, true
}

func moveFile(dst, orig string) error {
	file, err := os.Open(orig)
	if err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dstFile, file); err != nil {
		return err
	}

	if err := os.Remove(orig); err != nil {
		return err
	}

	return nil
}
