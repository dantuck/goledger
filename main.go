package main

import "os"
import "fmt"
import "flag"

import "github.com/prataprc/golog"

var options struct {
	dbname   string
	journals []string
	loglevel string
}

func argparse() {
	var journals string

	f := flag.NewFlagSet("ledger", flag.ExitOnError)
	f.Usage = func() {
		fmsg := "Usage of command: %v [ARGS]\n"
		fmt.Printf(fmsg, os.Args[0])
		f.PrintDefaults()
	}

	f.StringVar(&journals, "f", "example/first.ldg",
		"comma separated list of input files")
	f.StringVar(&options.dbname, "db", "devjournal",
		"provide datastore name")
	f.StringVar(&options.loglevel, "log", "warn",
		"console log level")
	f.Parse(os.Args[1:])

	options.journals = Parsecsv(journals)

	return
}

func main() {
	argparse()

	logsetts := map[string]interface{}{
		"log.level":      options.loglevel,
		"log.file":       "",
		"log.timeformat": "",
		"log.prefix":     "[%v]",
	}
	log.SetLogger(nil, logsetts)

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("os.Getwd(): %v\n", err)
		os.Exit(1)
	}

	NewDatastore(options.dbname)

	journals := getjournals(cwd)
	journals = append(journals, options.journals...)
	for _, journal := range journals {
		log.Debugf("processing journal %q\n", journal)
		//firstpass(db, journal)
	}
}
