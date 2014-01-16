package rest

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"shelter/config"
	"shelter/net/http/rest/messages"
	"strconv"
)

// Function created to run the listeners before dropping privileges in the main binary, so
// that we can listen in low ports without keeping the program as a super user
func Listen() ([]net.Listener, error) {
	listeners := make([]net.Listener, 0, len(config.ShelterConfig.RESTServer.Listeners))

	for _, v := range config.ShelterConfig.RESTServer.Listeners {
		ipAndPort := net.JoinHostPort(v.IP, strconv.Itoa(v.Port))

		if v.TLS {
			certificatePath := fmt.Sprintf("%s/%s",
				config.ShelterConfig.BasePath,
				config.ShelterConfig.RESTServer.TLS.CertificatePath,
			)

			privateKeyPath := fmt.Sprintf("%s/%s",
				config.ShelterConfig.BasePath,
				config.ShelterConfig.RESTServer.TLS.PrivateKeyPath,
			)

			cert, err := tls.LoadX509KeyPair(certificatePath, privateKeyPath)
			if err != nil {
				return nil, err
			}

			tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}

			ln, err := tls.Listen("tcp", ipAndPort, &tlsConfig)
			if err != nil {
				return nil, err
			}

			listeners = append(listeners, ln)

		} else {
			ln, err := net.Listen("tcp", ipAndPort)
			if err != nil {
				return nil, err
			}

			listeners = append(listeners, ln)
		}
	}

	return listeners, nil
}

func Start(listeners []net.Listener) error {
	// Initialize language configuration file
	if err := loadMessages(); err != nil {
		return err
	}

	// Initialize CIDR whitelist
	if err := loadACL(); err != nil {
		return err
	}

	for _, v := range listeners {
		go http.Serve(v, mux)
	}

	return nil
}

func loadMessages() error {
	messagePath := fmt.Sprintf("%s/%s",
		config.ShelterConfig.BasePath,
		config.ShelterConfig.RESTServer.LanguageConfigPath,
	)

	if err := messages.LoadConfig(messagePath); err != nil {
		return err
	}

	return nil
}

func loadACL() error {
	for _, cidrStr := range config.ShelterConfig.RESTServer.ACL {
		if _, cidr, err := net.ParseCIDR(cidrStr); err == nil {
			mux.ACL = append(mux.ACL, cidr)
		} else {
			return err
		}
	}

	return nil
}
