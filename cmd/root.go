package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var notionToken string
var logLevel string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "notionbackup",
	Short: "A tool to backup and restore the Notion workspace",
	Long: "Notion Backup is a tool to take backup of whole Notion workspace or " +
		"a specific set of Pages or Databases and restore them back to different " +
		"or same Noion workspace.",
}

// Execute adds all child commands to the root command and sets flags
// appropriately. This is called by main.main().
// It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&notionToken, "token", "",
		"Notion integration token that will be used for fetching Notion objects "+
			"from Notion API. Alternatively, one can set Notion integration token "+
			"as environment variable 'NTN_TOKEN'.")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"Level of logging. (Log levels: info, debug, trace)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("ntn")
	viper.AutomaticEnv()
}

func validateNonEmptyNotionToken() {
	if notionToken == "" {
		notionToken = viper.GetString("token")
		if notionToken == "" {
			fmt.Fprintf(os.Stderr, "Please provide Notion secret token with "+
				"--token flag or export it as an environment variable 'NTN_TOKEN'.\n")
			os.Exit(1)
		}
	}
}

type timeHook struct{}

func (t timeHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	e.Time("time", time.Now())
}

func getLogger() (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		return zerolog.Logger{}, errors.Wrapf(err, "Couldn't parse log level")
	}

	out := os.Stderr
	writer := zerolog.ConsoleWriter{
		Out:        out,
		TimeFormat: time.RFC822,
	}

	if !term.IsTerminal(int(out.Fd())) {
		writer.NoColor = true
	}
	log := zerolog.New(writer).
		Hook(timeHook{}).
		Level(level)

	return log, nil
}
