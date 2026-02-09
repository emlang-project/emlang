package main

import (
	"fmt"
	"os"

	"github.com/emlang-project/emlang/internal/parser"
)

func main() {
	file, err := os.Open("tmp/anchor.yaml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	doc, err := parser.Parse(file)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Slices: %v\n", len(doc.Slices))
	for name, slice := range doc.Slices {
		fmt.Printf("  %s: %d elements\n", name, len(slice.Elements))
	}
}
