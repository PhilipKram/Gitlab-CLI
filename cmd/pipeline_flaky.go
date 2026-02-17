package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// FlakyJobStats represents statistics for a potentially flaky job.
type FlakyJobStats struct {
	JobName        string  `json:"job_name"`
	Stage          string  `json:"stage"`
	TotalRuns      int     `json:"total_runs"`
	SuccessCount   int     `json:"success_count"`
	FailureCount   int     `json:"failure_count"`
	FlakinessRate  float64 `json:"flakiness_rate"`
	LastFailure    string  `json:"last_failure,omitempty"`
	LastSuccess    string  `json:"last_success,omitempty"`
}

// FlakyJobsResult represents the result of the flaky jobs analysis.
type FlakyJobsResult struct {
	Jobs           []FlakyJobStats `json:"jobs"`
	TotalPipelines int             `json:"total_pipelines"`
	TotalJobs      int             `json:"total_jobs"`
	FlakyJobs      int             `json:"flaky_jobs"`
	TimePeriodDays int             `json:"time_period_days"`
	Branch         string          `json:"branch,omitempty"`
}

func newPipelineFlakyCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch    string
		days      int
		threshold float64
		limit     int
		format    string
		jsonFlag  bool
	)

	cmd := &cobra.Command{
		Use:   "flaky",
		Short: "Detect flaky jobs with inconsistent results",
		Long:  "Analyze job results across recent pipelines to identify jobs with inconsistent pass/fail patterns that may indicate flaky tests.",
		Example: `  $ glab pipeline flaky
  $ glab pipeline flaky --branch main --days 7
  $ glab pipeline flaky --threshold 0.2 --limit 20 --format json`,
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
				ListOptions:  gitlab.ListOptions{PerPage: 100},
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

			// Aggregate job results by job name
			jobStats := make(map[string]*flakyJobAggregator)
			totalJobs := 0

			for _, pipeline := range allPipelines {
				jobs, _, err := client.Jobs.ListPipelineJobs(project, pipeline.ID, nil)
				if err != nil {
					// Skip pipelines with job fetch errors, continue with others
					continue
				}

				for _, job := range jobs {
					totalJobs++
					key := job.Name
					if _, exists := jobStats[key]; !exists {
						jobStats[key] = &flakyJobAggregator{
							jobName:      job.Name,
							stage:        job.Stage,
							successCount: 0,
							failureCount: 0,
							lastSuccess:  "",
							lastFailure:  "",
						}
					}

					// Track success/failure counts
					if job.Status == "success" {
						jobStats[key].successCount++
						if job.FinishedAt != nil {
							jobStats[key].lastSuccess = job.FinishedAt.Format(time.RFC3339)
						}
					} else if job.Status == "failed" {
						jobStats[key].failureCount++
						if job.FinishedAt != nil {
							jobStats[key].lastFailure = job.FinishedAt.Format(time.RFC3339)
						}
					}
				}
			}

			if len(jobStats) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No jobs found in the specified pipelines")
				return nil
			}

			// Calculate flakiness statistics and filter flaky jobs
			var stats []FlakyJobStats
			for _, agg := range jobStats {
				totalRuns := agg.successCount + agg.failureCount

				// Skip jobs with no completed runs or only one type of result
				if totalRuns == 0 || agg.successCount == 0 || agg.failureCount == 0 {
					continue
				}

				// Calculate flakiness rate (percentage of failures)
				flakinessRate := float64(agg.failureCount) / float64(totalRuns) * 100

				// Filter by threshold (only show jobs above the threshold)
				if threshold > 0 && flakinessRate < threshold {
					continue
				}

				stats = append(stats, FlakyJobStats{
					JobName:       agg.jobName,
					Stage:         agg.stage,
					TotalRuns:     totalRuns,
					SuccessCount:  agg.successCount,
					FailureCount:  agg.failureCount,
					FlakinessRate: flakinessRate,
					LastFailure:   agg.lastFailure,
					LastSuccess:   agg.lastSuccess,
				})
			}

			if len(stats) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No flaky jobs detected in the specified time period")
				return nil
			}

			// Sort by flakiness rate (most flaky first - closest to 50% is most unpredictable)
			sort.Slice(stats, func(i, j int) bool {
				// Calculate distance from 50% (perfect flakiness)
				distI := abs(stats[i].FlakinessRate - 50.0)
				distJ := abs(stats[j].FlakinessRate - 50.0)
				return distI < distJ
			})

			// Limit results
			if limit > 0 && len(stats) > limit {
				stats = stats[:limit]
			}

			result := FlakyJobsResult{
				Jobs:           stats,
				TotalPipelines: len(allPipelines),
				TotalJobs:      totalJobs,
				FlakyJobs:      len(stats),
				TimePeriodDays: days,
				Branch:         branch,
			}

			return f.FormatAndPrint(result, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Filter by branch or tag")
	cmd.Flags().IntVarP(&days, "days", "d", 30, "Number of days to analyze")
	cmd.Flags().Float64VarP(&threshold, "threshold", "t", 0.0, "Minimum flakiness rate threshold (0-100, 0 shows all flaky jobs)")
	cmd.Flags().IntVarP(&limit, "limit", "L", 20, "Maximum number of flaky jobs to display")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

// flakyJobAggregator is a helper struct for aggregating job results.
type flakyJobAggregator struct {
	jobName      string
	stage        string
	successCount int
	failureCount int
	lastSuccess  string
	lastFailure  string
}

// abs returns the absolute value of a float64.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
