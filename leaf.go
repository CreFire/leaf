package leaf

import (
	"github.com/CreFire/leaf/cluster"
	"github.com/CreFire/leaf/conf"
	"github.com/CreFire/leaf/console"
	"github.com/CreFire/leaf/module"
	nested "github.com/antonfisher/nested-logrus-formatter"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/signal"
)

func Run(mods ...module.Module) {
	// logger
	if conf.LogLevel != 0 {
		log.SetLevel(log.Level(conf.LogLevel))
		log.SetFormatter(&nested.Formatter{
			FieldsOrder:           []string{"component", "category"},
			TimestampFormat:       "",
			HideKeys:              true,
			NoColors:              false,
			NoFieldsColors:        false,
			NoFieldsSpace:         false,
			ShowFullLevel:         false,
			NoUppercaseLevel:      false,
			TrimMessages:          false,
			CallerFirst:           false,
			CustomCallerFormatter: nil,
		})
		writeFile, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("create file log.txt failed: %v", err)
		}
		log.SetOutput(io.MultiWriter(writeFile, os.Stdout))
	}

	log.Info("Leaf %v starting up", version)

	// module
	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}
	module.Init()

	// cluster
	cluster.Init()

	// console
	console.Init()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	log.Info("Leaf closing down (signal: %v)", sig)
	console.Destroy()
	cluster.Destroy()
	module.Destroy()
}
