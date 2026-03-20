package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func newCILintCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		ref         string
		dryRun      bool
		includeJobs bool
		format      string
		jsonFlag    bool
	)

	cmd := &cobra.Command{
		Use:   "lint [<file>]",
		Short: "Validate CI/CD configuration",
		Long: `Validate a project's CI/CD configuration.

Without arguments, validates the project's committed .gitlab-ci.yml.
With a file argument (or stdin via -), validates the provided YAML content.`,
		Example: `  $ glab pipeline lint
  $ glab pipeline lint --ref main --dry-run
  $ glab pipeline lint .gitlab-ci.yml
  $ cat .gitlab-ci.yml | glab pipeline lint -
  $ glab pipeline lint --include-jobs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			var result *gitlab.ProjectLintResult
			var resp *gitlab.Response

			if len(args) > 0 {
				// File or stdin mode: validate provided YAML content
				var content string
				if args[0] == "-" {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("reading stdin: %w", err)
					}
					content = string(data)
				} else {
					data, err := os.ReadFile(args[0])
					if err != nil {
						return fmt.Errorf("reading file %s: %w", args[0], err)
					}
					content = string(data)
				}

				opts := &gitlab.ProjectNamespaceLintOptions{
					Content:     &content,
					DryRun:      &dryRun,
					IncludeJobs: &includeJobs,
				}
				if ref != "" {
					opts.Ref = &ref
				}

				result, resp, err = client.Validate.ProjectNamespaceLint(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/ci/lint"
					return errors.NewAPIError("POST", url, statusCode, "Failed to lint CI configuration", err)
				}
			} else {
				// Project mode: validate committed config
				opts := &gitlab.ProjectLintOptions{
					DryRun:      &dryRun,
					IncludeJobs: &includeJobs,
				}
				if ref != "" {
					opts.Ref = &ref
				}

				result, resp, err = client.Validate.ProjectLint(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/ci/lint"
					return errors.NewAPIError("GET", url, statusCode, "Failed to lint CI configuration", err)
				}
			}

			// JSON output
			outputFormat, fmtErr := f.ResolveFormat(format, jsonFlag)
			if fmtErr != nil {
				return fmtErr
			}
			if outputFormat == "json" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			// Human-readable output
			out := f.IOStreams.Out
			if result.Valid {
				_, _ = fmt.Fprintln(out, "CI configuration is valid")
			} else {
				_, _ = fmt.Fprintln(out, "CI configuration is invalid")
			}

			if len(result.Errors) > 0 {
				_, _ = fmt.Fprintln(out, "\nErrors:")
				for _, e := range result.Errors {
					_, _ = fmt.Fprintf(out, "  - %s\n", e)
				}
			}

			if len(result.Warnings) > 0 {
				_, _ = fmt.Fprintln(out, "\nWarnings:")
				for _, w := range result.Warnings {
					_, _ = fmt.Fprintf(out, "  - %s\n", w)
				}
			}

			if len(result.Includes) > 0 {
				_, _ = fmt.Fprintln(out, "\nIncludes:")
				for _, inc := range result.Includes {
					parts := []string{}
					if inc.Type != "" {
						parts = append(parts, inc.Type)
					}
					if inc.Location != "" {
						parts = append(parts, inc.Location)
					}
					_, _ = fmt.Fprintf(out, "  - %s\n", strings.Join(parts, ": "))
				}
			}

			if !result.Valid {
				return fmt.Errorf("CI configuration has %d error(s)", len(result.Errors))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&ref, "ref", "", "Branch or tag to use as context for linting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run pipeline creation simulation")
	cmd.Flags().BoolVar(&includeJobs, "include-jobs", false, "Include job details in the response")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}
