package main

import (
	"fmt"
	"os"
	"path/filepath"

	validator "github.com/idsulik/helm-cel/pkg/validator"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cel [flags] CHART",
	Short: "Validate Helm values using CEL expressions",
	Long: `A Helm plugin to validate values.yaml using CEL expressions defined in values.cel.yaml.
Example: helm cel ./mychart`,
	RunE:          runCelValidation,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
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

	v := validator.New()
	result, err := v.ValidateChart(absPath)

	if err != nil {
		return err
	}

	if result.HasErrors() {
		return result
	}

	if len(result.Warnings) > 0 {
		fmt.Println(result.Error())
		fmt.Println("-------------------------------------------------")
		fmt.Println("⚠️✅ Values validation successful with warnings!")
	} else {
		fmt.Println("✅ Values validation successful!")
	}

	return nil
}
