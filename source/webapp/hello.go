package main

import (
	"context"
	"os"
    "webapp/templates"
)

func main() {
    component := templates.Hello("John")
	component.Render(context.Background(), os.Stdout)
}
