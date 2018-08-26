package githosts

import (
	"log"
	"os"
)

const (
	workingDIRName  = ".working"
	bundleExtension = ".bundle"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "soba: ", log.Lshortfile|log.LstdFlags)
}

// Backup accepts a Git hosting provider and executes the backup task for it
func Backup(providerName, backupDIR string) {
	var provider gitProvider
	var err error
	switch providerName {
	case "bitbucket":
		input := newHostInput{
			ProviderName: "BitBucket",
			APIURL:       "https://api.bitbucket.org/1.0",
		}
		provider, err = createHost(input)
		if err != nil {
			logger.Fatal(err)
		}
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
}
