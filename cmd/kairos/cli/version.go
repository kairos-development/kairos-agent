package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long: `Display version, build, and runtime information.

Shows:
- Version number
- Git commit hash
- Build timestamp
- Go version
- OS/Architecture
- Build integrity status`,
		RunE: runVersion,
	}

	cmd.Flags().Bool("short", false, "Show only version number")

	return cmd
}

func runVersion(cmd *cobra.Command, args []string) error {
	short, _ := cmd.Flags().GetBool("short")

	if short {
		fmt.Println(Version)
		return nil
	}

	fmt.Printf("Kairos Trading Agent\n")
	fmt.Printf("═══════════════════════════════════════════════════════════\n")
	fmt.Printf("Version:      %s\n", Version)
	fmt.Printf("Commit:       %s\n", Commit)
	fmt.Printf("Built:        %s\n", BuildTime)
	fmt.Printf("Go version:   %s\n", runtime.Version())
	fmt.Printf("OS/Arch:      %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("═══════════════════════════════════════════════════════════\n")

	return nil
}
