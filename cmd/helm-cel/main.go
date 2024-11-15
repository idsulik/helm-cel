package main

import (
	"fmt"
	"os"
	"path/filepath"

	validator "github.com/idsulik/helm-cel/pkg/validation"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cel [flags] CHART",
	Short: "Validate Helm values using CEL expressions",
	Long: `A Helm plugin to validate values.yaml using CEL expressions defined in values.cel.yaml.
Example: helm cel ./mychart`,
	RunE: runCelValidation,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runCelValidation(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("chart path is required")
	}

	chartPath := args[0]
	absPath, err := filepath.Abs(chartPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	validator := validator.New()
	if err := validator.ValidateChart(absPath); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	fmt.Println("✅ Values validation successful!")
	return nil
}
