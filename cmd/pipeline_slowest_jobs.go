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

// JobDurationStats represents aggregated job duration statistics.
type JobDurationStats struct {
	JobName         string  `json:"job_name"`
	Stage           string  `json:"stage"`
	TotalRuns       int     `json:"total_runs"`
	AverageDuration float64 `json:"average_duration"`
	MinDuration     float64 `json:"min_duration"`
	MaxDuration     float64 `json:"max_duration"`
	TotalDuration   float64 `json:"total_duration"`
}

// SlowestJobsResult represents the result of the slowest jobs analysis.
type SlowestJobsResult struct {
	Jobs           []JobDurationStats `json:"jobs"`
	TotalPipelines int                `json:"total_pipelines"`
	TotalJobs      int                `json:"total_jobs"`
	TimePeriodDays int                `json:"time_period_days"`
	Branch         string             `json:"branch,omitempty"`
}

func newPipelineSlowestJobsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch   string
		days     int
		limit    int
		format   string
		jsonFlag bool
	)

	cmd := &cobra.Command{
		Use:   "slowest-jobs",
		Short: "Identify slowest CI jobs",
		Long:  "Analyze job durations across recent pipelines and identify the slowest jobs by average duration.",
		Example: `  $ glab pipeline slowest-jobs
  $ glab pipeline slowest-jobs --branch main --days 7
  $ glab pipeline slowest-jobs --limit 20 --format json`,
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

			// Aggregate job durations by job name
			jobStats := make(map[string]*jobAggregator)
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
						jobStats[key] = &jobAggregator{
							jobName:   job.Name,
							stage:     job.Stage,
							durations: []float64{},
						}
					}
					jobStats[key].durations = append(jobStats[key].durations, job.Duration)
				}
			}

			if len(jobStats) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No jobs found in the specified pipelines")
				return nil
			}

			// Calculate statistics and convert to slice
			var stats []JobDurationStats
			for _, agg := range jobStats {
				if len(agg.durations) == 0 {
					continue
				}

				var total, min, max float64
				min = agg.durations[0]
				max = agg.durations[0]

				for _, d := range agg.durations {
					total += d
					if d < min {
						min = d
					}
					if d > max {
						max = d
					}
				}

				avg := total / float64(len(agg.durations))

				stats = append(stats, JobDurationStats{
					JobName:         agg.jobName,
					Stage:           agg.stage,
					TotalRuns:       len(agg.durations),
					AverageDuration: avg,
					MinDuration:     min,
					MaxDuration:     max,
					TotalDuration:   total,
				})
			}

			// Sort by average duration (slowest first)
			sort.Slice(stats, func(i, j int) bool {
				return stats[i].AverageDuration > stats[j].AverageDuration
			})

			// Limit results
			if limit > 0 && len(stats) > limit {
				stats = stats[:limit]
			}

			result := SlowestJobsResult{
				Jobs:           stats,
				TotalPipelines: len(allPipelines),
				TotalJobs:      totalJobs,
				TimePeriodDays: days,
				Branch:         branch,
			}

			return f.FormatAndPrint(result, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Filter by branch or tag")
	cmd.Flags().IntVarP(&days, "days", "d", 30, "Number of days to analyze")
	cmd.Flags().IntVarP(&limit, "limit", "L", 10, "Maximum number of jobs to display")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

// jobAggregator is a helper struct for aggregating job durations.
type jobAggregator struct {
	jobName   string
	stage     string
	durations []float64
}
