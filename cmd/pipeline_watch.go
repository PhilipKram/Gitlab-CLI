package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func statusColor(status string) string {
	switch status {
	case "success":
		return "\033[32m" + status + "\033[0m" // green
	case "failed":
		return "\033[31m" + status + "\033[0m" // red
	case "running", "pending":
		return "\033[33m" + status + "\033[0m" // yellow
	case "canceled":
		return "\033[90m" + status + "\033[0m" // gray
	default:
		return status
	}
}

func isTerminalStatus(status string) bool {
	switch status {
	case "success", "failed", "canceled", "skipped":
		return true
	default:
		return false
	}
}

func newPipelineWatchCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		interval time.Duration
		failFast bool
		jobsOnly []string
	)

	cmd := &cobra.Command{
		Use:   "watch <pipeline-id>",
		Short: "Watch a pipeline in real-time",
		Long:  "Poll a pipeline and its jobs at a regular interval, displaying status updates until the pipeline reaches a terminal state.",
		Example: `  $ glab pipeline watch 12345
  $ glab pipeline watch 12345 --interval 10s
  $ glab pipeline watch 12345 --fail-fast
  $ glab pipeline watch 12345 --jobs-only build,test`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid pipeline ID %q: must be an integer", args[0])
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			out := f.IOStreams.Out

			// Set up signal handling for graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				cancel()
			}()

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			// Poll immediately on first iteration, then on ticker
			first := true
			for {
				if !first {
					select {
					case <-ctx.Done():
						_, _ = fmt.Fprintln(out, "\nWatch canceled.")
						return nil
					case <-ticker.C:
					}
				}
				first = false

				pipeline, resp, err := client.Pipelines.GetPipeline(project, pipelineID)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					return errors.NewAPIError(
						"GET",
						fmt.Sprintf("projects/%s/pipelines/%d", project, pipelineID),
						statusCode,
						"Failed to get pipeline",
						err,
					)
				}

				jobs, _, err := client.Jobs.ListPipelineJobs(project, pipelineID, nil)
				if err != nil {
					// Non-fatal: continue without jobs
					jobs = nil
				}

				// Clear screen
				_, _ = fmt.Fprint(out, "\033[2J\033[H")

				// Pipeline header
				_, _ = fmt.Fprintf(out, "Pipeline #%d  %s\n", pipeline.ID, statusColor(pipeline.Status))
				_, _ = fmt.Fprintf(out, "Ref:       %s\n", pipeline.Ref)
				_, _ = fmt.Fprintf(out, "Source:    %s\n", pipeline.Source)
				if pipeline.CreatedAt != nil {
					_, _ = fmt.Fprintf(out, "Created:   %s\n", pipeline.CreatedAt.Format(time.RFC3339))
				}
				if pipeline.Duration > 0 {
					_, _ = fmt.Fprintf(out, "Duration:  %ds\n", pipeline.Duration)
				}
				_, _ = fmt.Fprintln(out)

				// Filter jobs if --jobs-only is set
				displayJobs := jobs
				if len(jobsOnly) > 0 && len(jobs) > 0 {
					var filtered []*gitlab.Job
					for _, job := range jobs {
						for _, pattern := range jobsOnly {
							if strings.Contains(job.Name, pattern) {
								filtered = append(filtered, job)
								break
							}
						}
					}
					displayJobs = filtered
				}

				// Jobs table
				if len(displayJobs) > 0 {
					_, _ = fmt.Fprintf(out, "%-30s %-20s %-12s %s\n", "NAME", "STAGE", "STATUS", "DURATION")
					_, _ = fmt.Fprintf(out, "%-30s %-20s %-12s %s\n", "----", "-----", "------", "--------")
					for _, job := range displayJobs {
						duration := ""
						if job.Duration > 0 {
							duration = fmt.Sprintf("%.0fs", job.Duration)
						}
						_, _ = fmt.Fprintf(out, "%-30s %-20s %-12s %s\n",
							truncateWatch(job.Name, 30),
							truncateWatch(job.Stage, 20),
							statusColor(job.Status),
							duration,
						)
					}
				}

				// Check for early exit on job failure
				if failFast {
					watchedJobs := displayJobs
					if len(jobsOnly) == 0 {
						watchedJobs = jobs
					}
					for _, job := range watchedJobs {
						if job.Status == "failed" {
							_, _ = fmt.Fprintf(out, "\nJob %q failed — exiting (--fail-fast)\n", job.Name)
							return fmt.Errorf("job %q in pipeline #%d failed", job.Name, pipeline.ID)
						}
					}
				}

				if isTerminalStatus(pipeline.Status) {
					_, _ = fmt.Fprintf(out, "\nPipeline finished with status: %s\n", statusColor(pipeline.Status))
					if pipeline.Status == "failed" {
						return fmt.Errorf("pipeline #%d failed", pipeline.ID)
					}
					return nil
				}
			}
		},
	}

	cmd.Flags().DurationVarP(&interval, "interval", "i", 5*time.Second, "Polling interval (e.g. 5s, 10s)")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Exit immediately when any job fails")
	cmd.Flags().StringSliceVar(&jobsOnly, "jobs-only", nil, "Filter displayed/watched jobs by substring match (comma-separated)")

	return cmd
}

func truncateWatch(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
