// Command modhaven is a terminal save editor for Space Haven.
//
// Run it with no arguments for the interactive interface, or give it a command
// to script the same edits.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/SecondPort/mod_space_haven/internal/catalog"
	"github.com/SecondPort/mod_space_haven/internal/cli"
	"github.com/SecondPort/mod_space_haven/internal/tui"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	var (
		lang        string
		dir         string
		save        string
		showVersion bool
	)

	fs := flag.NewFlagSet("modhaven", flag.ContinueOnError)
	cli.Flags(fs, &lang, &dir, &save)
	fs.BoolVar(&showVersion, "version", false, "print the version and exit")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "modhaven — Space Haven save editor\n\nUsage:\n  modhaven [flags] [command]\n\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n%s\n", cli.Usage)
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if showVersion {
		fmt.Println("modhaven", version)
		return
	}

	language := catalog.ParseLanguage(lang)

	if args := fs.Args(); len(args) > 0 {
		os.Exit(cli.Run(cli.Config{
			Language: language,
			SavesDir: dir,
			SavePath: save,
			Out:      os.Stdout,
			Err:      os.Stderr,
		}, args))
	}

	cat, err := catalog.Embedded()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := tui.Run(cat, tui.Options{
		Language: language,
		SavesDir: dir,
		SavePath: save,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
