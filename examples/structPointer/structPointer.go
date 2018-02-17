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
	ExampleFoo   *Foo
}

// Foo is a struct refered in config
type Foo struct {
	Name string `env:"EXAMPLE_FOO"`
}

func main() {
	cfg := &config{ExampleFoo: &Foo{Name: "a"}}

	// Parse for built-in types
	if err := env.Parse(cfg); err != nil {
		log.Fatal("Unable to parse envs: ", err)
	}

	fmt.Printf("%+v\n", cfg)
	fmt.Printf("ExampleFoo: %+v\n", *cfg.ExampleFoo)
}
