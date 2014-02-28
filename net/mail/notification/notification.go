package notification

import (
	"bytes"
	"fmt"
	"github.com/rafaeljusto/shelter/config"
	"github.com/rafaeljusto/shelter/dao"
	"github.com/rafaeljusto/shelter/database/mongodb"
	"github.com/rafaeljusto/shelter/log"
	"github.com/rafaeljusto/shelter/model"
	"net/smtp"
)

// Notify is responsable for selecting the domains that should be notified in the system.
// It will send alert e-mails for each owner of a domain
func Notify() {
	defer func() {
		// Something went really wrong while notifying the owners. Log the error stacktrace
		// and move out
		if r := recover(); r != nil {
			log.Println("Panic detected while notifying the owners. Details:", r)
		}
	}()

	database, databaseSession, err := mongodb.Open(
		config.ShelterConfig.Database.URI,
		config.ShelterConfig.Database.Name,
	)

	if err != nil {
		log.Println("Error while initializing database. Details:", err)
		return
	}
	defer databaseSession.Close()

	domainDAO := dao.DomainDAO{
		Database: database,
	}

	domainChannel, err := domainDAO.FindAllAsyncToBeNotified(
		config.ShelterConfig.Notification.NameserverErrorAlertDays,
		config.ShelterConfig.Notification.NameserverTimeoutAlertDays,
		config.ShelterConfig.Notification.DSErrorAlertDays,
		config.ShelterConfig.Notification.DSTimeoutAlertDays,

		// TODO: Should we move this configuration parameter to a place were both modules can
		// access it. This sounds better for configuration deployment
		config.ShelterConfig.Scan.VerificationIntervals.MaxExpirationAlertDays,
	)

	if err != nil {
		log.Println("Error retrieving domains to notify. Details:", err)
		return
	}

	// Dispatch the asynchronous part of the method
	for {
		// Get domain from the database (one-by-one)
		domainResult := <-domainChannel

		// Detect errors while retrieving a specific domain. We are not going to stop all the
		// process when only one domain got an error
		if domainResult.Error != nil {
			log.Println("Error retrieving domain to notify. Details:", domainResult.Error)
			continue
		}

		// Problem detected while retrieving a domain or we don't have domains anymore
		if domainResult.Error != nil || domainResult.Domain == nil {
			break
		}

		if err := notifyDomain(domainResult.Domain); err != nil {
			log.Println("Error notifying a domain. Details:", err)
		}
	}
}

// Function used to notify a single domain. It can return error if there's a problem while
// filling the template or sending the e-mail
func notifyDomain(domain *model.Domain) error {
	from := config.ShelterConfig.Notification.From

	emailsPerLanguage := make(map[string][]string)
	for _, owner := range domain.Owners {
		emailsPerLanguage[owner.Language] =
			append(emailsPerLanguage[owner.Language], owner.Email.String())
	}

	server := fmt.Sprintf("%s:%d",
		config.ShelterConfig.Notification.SMTPServer.Server,
		config.ShelterConfig.Notification.SMTPServer.Port,
	)

	for language, emails := range emailsPerLanguage {
		t := getTemplate(language)

		var msg bytes.Buffer
		if err := t.Execute(&msg, domain); err != nil {
			return err
		}

		switch config.ShelterConfig.Notification.SMTPServer.Auth.Type {
		case config.AuthenticationTypePlain:
			auth := smtp.PlainAuth("",
				config.ShelterConfig.Notification.SMTPServer.Auth.Username,
				config.ShelterConfig.Notification.SMTPServer.Auth.Password,
				config.ShelterConfig.Notification.SMTPServer.Server,
			)

			if err := smtp.SendMail(server, auth, from, emails, msg.Bytes()); err != nil {
				return err
			}

		case config.AuthenticationTypeCRAMMD5Auth:
			auth := smtp.CRAMMD5Auth(
				config.ShelterConfig.Notification.SMTPServer.Auth.Username,
				config.ShelterConfig.Notification.SMTPServer.Auth.Password,
			)

			if err := smtp.SendMail(server, auth, from, emails, msg.Bytes()); err != nil {
				return err
			}

		default:
			if err := smtp.SendMail(server, nil, from, emails, msg.Bytes()); err != nil {
				return err
			}
		}
	}

	return nil
}
