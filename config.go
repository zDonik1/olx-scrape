package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfg = Config{}

type Config struct {
	Verbose           bool   `mapstructure:"verbose"`
	Jobs              uint   `mapstructure:"jobs"`
	CacheDir          string `mapstructure:"cache-dir"`
	RefreshCache      bool   `mapstructure:"refresh-cache"`
	RefreshPagesCache bool   `mapstructure:"refresh-pages-cache"`
	RefreshDataCache  bool   `mapstructure:"refresh-data-cache"`
	AiProcessing      bool   `mapstructure:"ai-processing"`
	Category          string `mapstructure:"category"`
	Pages             uint   `mapstructure:"pages"`
	MaxAds            uint   `mapstructure:"max-ads"`
}

func initConfig() {
	viper := viper.New()
	flags := pflag.NewFlagSet("config", pflag.ContinueOnError)

	flags.BoolP("verbose", "v", false, "print verbose output")
	flags.UintP("jobs", "j", 10, "parallel jobs to spawn for fetching ad data")
	flags.String("cache-dir", "./cache", "path to cache directory")
	flags.BoolP("refresh-cache", "R", false, "invalidate and rebuild cache")
	flags.BoolP("refresh-pages-cache", "P", false, "invalidate and rebuild ad browser pages cache")
	flags.BoolP("refresh-data-cache", "D", false, "invalidate and rebuild ad data cache")
	flags.BoolP("ai-processing", "a", false, "enable AI processing")
	flags.StringP("category", "c", "", "category to scrape (example 'elektronika/kompyutery/nastolnye')")
	flags.UintP("pages", "p", 1, "pages to scan")
	flags.Uint("max-ads", 0, "maximum number of ads to process, 0 means no max")

	if err := viper.BindPFlags(flags); err != nil {
		slog.Error("failed to bind flags", "error", err)
		os.Exit(1)
	}

	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			os.Exit(0)
		}
		slog.Error("failed to parse flags", "error", err)
		os.Exit(1)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		slog.Error("error unmarshaling config", "error", err)
		os.Exit(1)
	}
}
