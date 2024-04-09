package main

import (
	"os"

	"github.com/daanvanberkel/fireflyiiibunq/bunq"
	"github.com/daanvanberkel/fireflyiiibunq/firefly"
	"github.com/daanvanberkel/fireflyiiibunq/sync"
	"github.com/daanvanberkel/fireflyiiibunq/util"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.Level = logrus.InfoLevel
	log.Out = os.Stdout

	config, err := util.LoadConfig()
	if err != nil {
		panic(err)
	}

	fireflyClient, err := firefly.NewFireflyClient(config, log)
	if err != nil {
		panic(err)
	}

	bunqClient, err := bunq.NewBunqClient(config, log)
	if err != nil {
		panic(err)
	}

	sync.StartPullSync(fireflyClient, bunqClient, log)
}
