package githosts

import (
	"log"
	"os"
)

const workingDIRName = ".working"

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "soba: ", log.Lshortfile|log.LstdFlags)
}

func Backup(providerName, backupDIR string) {
	var provider gitProvider
	var err error
	switch providerName {
	case "github":
		input := newHostInput{
			ProviderName: "Github",
			APIURL:       "https://api.github.com/graphql",
		}
		provider, err = createHost(input)
		if err != nil {
			logger.Fatal(err)
		}
	case "gitlab":
		input := newHostInput{
			ProviderName: "Gitlab",
			APIURL:       "https://gitlab.com/api/v4",
		}
		provider, err = createHost(input)
		if err != nil {
			logger.Fatal(err)
		}
	}
	provider.Backup(backupDIR)
	logger.Println("cleaning up")
	deleteDirIfExists(backupDIR + string(os.PathSeparator) + workingDIRName)
}
