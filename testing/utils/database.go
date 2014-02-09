package utils

import (
	"github.com/rafaeljusto/shelter/dao"
	"labix.org/v2/mgo"
)

// Function created to remove all entries from the database to ensure that the tests
// enviroments are always equal
func ClearDatabase(database *mgo.Database) {
	domainDAO := dao.DomainDAO{
		Database: database,
	}
	domainDAO.RemoveAll()

	scanDAO := dao.ScanDAO{
		Database: database,
	}
	scanDAO.RemoveAll()
}