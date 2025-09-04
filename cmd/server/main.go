package main

import (
	"fmt"

	"github.com/elidor/dungeogo/config"
)

func main() {
	cfg := config.NewConfig(config.NewFileProvider(".env"))
	fmt.Printf(cfg.GetValue(config.Port))
}
