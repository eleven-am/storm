package orm_generator

import (
	"fmt"

	"github.com/spf13/cobra"
)

// CLICommands provides ORM generation CLI commands
type CLICommands struct {
	// No database connection needed for code generation
}

func NewCLICommands() *CLICommands {
	return &CLICommands{}
}

func (cli *CLICommands) GetRootCommand() *cobra.Command {
	ormCmd := &cobra.Command{
		Use:   "orm",
		Short: "ORM code generation for type-safe database operations",
		Long:  `Generate type-safe ORM code from Go struct definitions including column constants, repositories, and query builders`,
	}

	ormCmd.AddCommand(cli.getValidateCommand())
	ormCmd.AddCommand(cli.getGenerateORMCommand())

	return ormCmd
}

func (cli *CLICommands) getValidateCommand() *cobra.Command {
	var packagePath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate ORM model definitions and relationships",
		Long: `Validate ORM model struct definitions, including:
- Struct tags (db, dbdef, orm)
- Primary key requirements
- Relationship definitions
- Foreign key references
- Field type compatibility`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Validating ORM models in %s...\n", packagePath)

			result := ValidateModelsFromDirectory(packagePath)

			if result.Valid {
				fmt.Printf("✓ All discovered models are valid\n")
				return nil
			}

			fmt.Printf("✗ Validation failed with %d errors:\n", len(result.Errors))
			for _, err := range result.Errors {
				fmt.Printf("  - %s\n", err.Error())
			}

			return fmt.Errorf("model validation failed")
		},
	}

	cmd.Flags().StringVar(&packagePath, "package", "./internal/db", "Package path containing model definitions")
	return cmd
}

func (cli *CLICommands) getGenerateORMCommand() *cobra.Command {
	var packagePath string
	var packageName string

	cmd := &cobra.Command{
		Use:   "generate-orm",
		Short: "Generate type-safe ORM code from model definitions",
		Long: `Auto-discover model structs and generate type-safe ORM code including:
- Type-safe column constants (Users.Name.Eq())
- Repository implementations with optimized queries
- Query builders with compile-time safety
- Relationship helpers for joins and includes

Code is generated in the same directory as the input models.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Generating type-safe ORM code from %s...\n", packagePath)

			config := GenerationConfig{
				PackageName: packageName,
				OutputDir:   packagePath,
			}

			generator := NewCodeGenerator(config)

			if err := generator.DiscoverModels(packagePath); err != nil {
				return fmt.Errorf("failed to discover models: %w", err)
			}

			if err := generator.GenerateAll(); err != nil {
				return fmt.Errorf("failed to generate code: %w", err)
			}

			modelNames := generator.GetModelNames()
			fmt.Printf("✓ Generated type-safe ORM code for %d models: %v\n", len(modelNames), modelNames)
			fmt.Printf("✓ Output written to %s\n", packagePath)

			return nil
		},
	}

	cmd.Flags().StringVar(&packagePath, "package", "./internal/db", "Package path containing model definitions")
	cmd.Flags().StringVar(&packageName, "pkg-name", "", "Package name for generated code (default: auto-detect from models)")

	return cmd
}
