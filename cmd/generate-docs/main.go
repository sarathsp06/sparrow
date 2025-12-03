package main

import (
	"fmt"
	"os"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// Tool to generate template functions documentation
func main() {
	// Generate and save the template functions documentation
	docs := client.GetFunctionDocumentation()

	err := os.WriteFile("TEMPLATE_FUNCTIONS.md", []byte(docs), 0644)
	if err != nil {
		fmt.Printf("Error writing documentation: %v\n", err)
		return
	}

	fmt.Println("Template functions documentation generated: TEMPLATE_FUNCTIONS.md")

	// List all available functions
	fmt.Println("\nAvailable template functions:")
	functions := client.ListAvailableFunctions()
	for name := range functions {
		fmt.Printf("- %s\n", name)
	}
}
