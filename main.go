package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	gromit, err := NewGromit(WithPromptPrefix("⚡️🤖"))
	if err != nil {
		fmt.Println("Error instantiating Gromit: ", err)
		os.Exit(1)
	}
	if err := gromit.Run(context.Background(), os.Args); err != nil {
		fmt.Println("Error running Gromit: ", err)
		os.Exit(1)
	}
}
