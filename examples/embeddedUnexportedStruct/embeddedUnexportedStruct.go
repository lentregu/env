package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lentregu/env"
)

type config struct {
	Home         string        `env:"HOME"`
	Port         int           `env:"PORT" envDefault:"3000"`
	IsProduction bool          `env:"PRODUCTION"`
	Hosts        []string      `env:"HOSTS" envSeparator:":"`
	Duration     time.Duration `env:"DURATION"`
	foo
}

type foo struct {
	Name1 string `env:"EXAMPLE_FOO1"`
	Name2 string `env:"EXAMPLE_FOO2"`
}

// In this example Foo is embedded in config
func main() {
	cfg := config{}

	// Parse for built-in types
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Unable to parse envs: ", err)
	}

	fmt.Printf("%+v\n", cfg)
}
