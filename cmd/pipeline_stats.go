package cmd

import (
	"fmt"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// PipelineStats represents pipeline statistics.
type PipelineStats struct {
	TotalPipelines int     `json:"total_pipelines"`
	SuccessCount   int     `json:"success_count"`
	FailedCount    int     `json:"failed_count"`
	CanceledCount  int     `json:"canceled_count"`
	SkippedCount   int     `json:"skipped_count"`
	RunningCount   int     `json:"running_count"`
	PendingCount   int     `json:"pending_count"`
	SuccessRate    float64 `json:"success_rate"`
	FailureRate    float64 `json:"failure_rate"`
	TimePeriodDays int     `json:"time_period_days"`
	Branch         string  `json:"branch,omitempty"`
}

func newPipelineStatsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch   string
		days     int
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show pipeline statistics",
		Long:  "Display pipeline success/failure rates and statistics for a configurable time period.",
		Example: `  $ glab pipeline stats
  $ glab pipeline stats --branch main --days 7
  $ glab pipeline stats --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			// Calculate date cutoff
			cutoffDate := time.Now().AddDate(0, 0, -days)

			// Fetch all pipelines within the time period
			opts := &gitlab.ListProjectPipelinesOptions{
				ListOptions: gitlab.ListOptions{PerPage: 100},
				UpdatedAfter: &cutoffDate,
			}

			if branch != "" {
				opts.Ref = &branch
			}

			var allPipelines []*gitlab.PipelineInfo
			page := 1

			for {
				opts.Page = int64(page)
				pipelines, resp, err := client.Pipelines.ListProjectPipelines(project, opts)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/projects/" + project + "/pipelines"
					return errors.NewAPIError("GET", url, statusCode, "Failed to list pipelines", err)
				}

				if len(pipelines) == 0 {
					break
				}

				allPipelines = append(allPipelines, pipelines...)

				// Check if we've retrieved all pages
				if resp.NextPage == 0 {
					break
				}
				page = int(resp.NextPage)
			}

			if len(allPipelines) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No pipelines found in the specified time period")
				return nil
			}

			// Calculate statistics
			stats := PipelineStats{
				TotalPipelines: len(allPipelines),
				TimePeriodDays: days,
				Branch:         branch,
			}

			for _, pipeline := range allPipelines {
				switch pipeline.Status {
				case "success":
					stats.SuccessCount++
				case "failed":
					stats.FailedCount++
				case "canceled":
					stats.CanceledCount++
				case "skipped":
					stats.SkippedCount++
				case "running":
					stats.RunningCount++
				case "pending":
					stats.PendingCount++
				}
			}

			// Calculate rates
			if stats.TotalPipelines > 0 {
				stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalPipelines) * 100
				stats.FailureRate = float64(stats.FailedCount) / float64(stats.TotalPipelines) * 100
			}

			return f.FormatAndPrint(stats, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Filter by branch or tag")
	cmd.Flags().IntVarP(&days, "days", "d", 30, "Number of days to analyze")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}
