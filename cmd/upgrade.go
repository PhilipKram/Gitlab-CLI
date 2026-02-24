package cmd

import (
	"fmt"
	"os"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/prompt"
	"github.com/PhilipKram/gitlab-cli/internal/update"
	"github.com/spf13/cobra"
)

// NewUpgradeCmd creates the upgrade command.
func NewUpgradeCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		checkOnly bool
		yes       bool
		force     bool
	)

	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"update"},
		Short:   "Upgrade glab to the latest version",
		Long:    "Check for and install the latest version of glab.",
		Example: `  $ glab upgrade
  $ glab upgrade --check
  $ glab upgrade --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := f.Version
			out := f.IOStreams.Out

			// Reject dev builds unless --force
			if version == "dev" && !force {
				return fmt.Errorf("cannot upgrade a development build\nUse --force to override, or download a release from:\nhttps://github.com/PhilipKram/Gitlab-CLI/releases")
			}

			// Detect install method
			method := update.DetectInstallMethod()
			switch method {
			case update.InstallBrew:
				fmt.Fprintln(out, "glab was installed via Homebrew. To upgrade, run:")
				fmt.Fprintln(out, "  brew upgrade glab")
				return nil
			case update.InstallDeb:
				fmt.Fprintln(out, "glab was installed via a Debian package. To upgrade, download the latest .deb from:")
				fmt.Fprintln(out, "  https://github.com/PhilipKram/Gitlab-CLI/releases")
				return nil
			case update.InstallRPM:
				fmt.Fprintln(out, "glab was installed via an RPM package. To upgrade, download the latest .rpm from:")
				fmt.Fprintln(out, "  https://github.com/PhilipKram/Gitlab-CLI/releases")
				return nil
			}

			// Check for latest version
			fmt.Fprintln(out, "Checking for updates...")
			result, err := update.CheckLatestRelease(version)
			if err != nil {
				return fmt.Errorf("failed to check for updates: %w\nPlease check your internet connection and try again", err)
			}

			if !result.IsNewer && !force {
				fmt.Fprintf(out, "glab is already up to date (v%s)\n", version)
				return nil
			}

			fmt.Fprintf(out, "Current version: v%s\n", version)
			fmt.Fprintf(out, "Latest version:  v%s\n", result.LatestVersion)

			if checkOnly {
				fmt.Fprintf(out, "\nRun `glab upgrade` to install the update.\n")
				return nil
			}

			// Confirm with user
			if !yes {
				confirmed, err := prompt.Confirm(f.IOStreams.In, out,
					fmt.Sprintf("Upgrade glab to v%s?", result.LatestVersion), true)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(out, "Upgrade cancelled.")
					return nil
				}
			}

			// Find the right asset
			archiveName := update.ArchiveName(result.LatestVersion)
			archiveURL, checksumURL, err := update.FindAssetURLs(result.Release, archiveName)
			if err != nil {
				return err
			}

			// Create temp directory for download
			tmpDir, err := os.MkdirTemp("", "glab-upgrade-*")
			if err != nil {
				return fmt.Errorf("creating temp directory: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			// Download archive
			fmt.Fprintf(out, "Downloading %s...\n", archiveName)
			archivePath, err := update.DownloadAsset(archiveURL, tmpDir)
			if err != nil {
				return err
			}

			// Verify checksum
			fmt.Fprintln(out, "Verifying checksum...")
			if err := update.VerifyChecksum(archivePath, checksumURL); err != nil {
				return err
			}

			// Extract binary
			fmt.Fprintln(out, "Extracting binary...")
			binaryPath, err := update.ExtractBinary(archivePath, tmpDir)
			if err != nil {
				return err
			}

			// Replace binary
			fmt.Fprintln(out, "Installing...")
			if err := update.ReplaceBinary(binaryPath); err != nil {
				return err
			}

			fmt.Fprintf(out, "Successfully upgraded glab to v%s\n", result.LatestVersion)
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&force, "force", false, "Force upgrade even for dev builds or when up-to-date")

	return cmd
}
