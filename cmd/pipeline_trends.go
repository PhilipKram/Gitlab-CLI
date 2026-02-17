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

// TrendBucket represents pipeline durations for a specific time period.
type TrendBucket struct {
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
	PipelineCount   int       `json:"pipeline_count"`
	AverageDuration float64   `json:"average_duration"`
	MinDuration     float64   `json:"min_duration"`
	MaxDuration     float64   `json:"max_duration"`
	TotalDuration   float64   `json:"total_duration"`
}

// PipelineTrends represents pipeline duration trends over time.
type PipelineTrends struct {
	Buckets         []TrendBucket `json:"buckets"`
	OverallTrend    string        `json:"overall_trend"`
	TrendPercentage float64       `json:"trend_percentage"`
	TotalPipelines  int           `json:"total_pipelines"`
	TimePeriodDays  int           `json:"time_period_days"`
	BucketSizeDays  int           `json:"bucket_size_days"`
	Branch          string        `json:"branch,omitempty"`
}

func newPipelineTrendsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		branch     string
		days       int
		bucketSize int
		format     string
		jsonFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "trends",
		Short: "Show pipeline duration trends",
		Long:  "Analyze pipeline duration trends over time, showing whether pipeline durations are increasing, decreasing, or stable.",
		Example: `  $ glab pipeline trends
  $ glab pipeline trends --branch main --days 14
  $ glab pipeline trends --bucket-size 3 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			// Validate bucket size
			if bucketSize < 1 {
				return fmt.Errorf("bucket-size must be at least 1 day")
			}
			if bucketSize > days {
				return fmt.Errorf("bucket-size (%d) cannot be larger than time period (%d days)", bucketSize, days)
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

			// Fetch full pipeline details to get duration information
			var pipelinesWithDuration []*gitlab.Pipeline
			for _, pipelineInfo := range allPipelines {
				pipeline, _, err := client.Pipelines.GetPipeline(project, pipelineInfo.ID)
				if err != nil {
					// Skip pipelines that fail to fetch, continue with others
					continue
				}
				pipelinesWithDuration = append(pipelinesWithDuration, pipeline)
			}

			if len(pipelinesWithDuration) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No pipeline details could be fetched")
				return nil
			}

			// Create time buckets
			numBuckets := (days + bucketSize - 1) / bucketSize // ceiling division
			buckets := make([]TrendBucket, numBuckets)
			now := time.Now()

			for i := 0; i < numBuckets; i++ {
				buckets[i] = TrendBucket{
					StartDate: now.AddDate(0, 0, -days+(i*bucketSize)),
					EndDate:   now.AddDate(0, 0, -days+((i+1)*bucketSize)),
				}
			}

			// Assign pipelines to buckets
			for _, pipeline := range pipelinesWithDuration {
				if pipeline.CreatedAt == nil || pipeline.Duration == 0 {
					continue
				}

				for i := range buckets {
					if (pipeline.CreatedAt.Equal(buckets[i].StartDate) || pipeline.CreatedAt.After(buckets[i].StartDate)) &&
						pipeline.CreatedAt.Before(buckets[i].EndDate) {
						buckets[i].PipelineCount++
						buckets[i].TotalDuration += float64(pipeline.Duration)

						if buckets[i].MinDuration == 0 || float64(pipeline.Duration) < buckets[i].MinDuration {
							buckets[i].MinDuration = float64(pipeline.Duration)
						}
						if float64(pipeline.Duration) > buckets[i].MaxDuration {
							buckets[i].MaxDuration = float64(pipeline.Duration)
						}
						break
					}
				}
			}

			// Calculate averages and filter empty buckets
			var nonEmptyBuckets []TrendBucket
			for _, bucket := range buckets {
				if bucket.PipelineCount > 0 {
					bucket.AverageDuration = bucket.TotalDuration / float64(bucket.PipelineCount)
					nonEmptyBuckets = append(nonEmptyBuckets, bucket)
				}
			}

			if len(nonEmptyBuckets) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No completed pipelines with duration found")
				return nil
			}

			// Sort buckets by start date
			sort.Slice(nonEmptyBuckets, func(i, j int) bool {
				return nonEmptyBuckets[i].StartDate.Before(nonEmptyBuckets[j].StartDate)
			})

			// Determine overall trend
			var overallTrend string
			var trendPercentage float64

			if len(nonEmptyBuckets) >= 2 {
				firstAvg := nonEmptyBuckets[0].AverageDuration
				lastAvg := nonEmptyBuckets[len(nonEmptyBuckets)-1].AverageDuration

				if firstAvg > 0 {
					trendPercentage = ((lastAvg - firstAvg) / firstAvg) * 100
				}

				// Classify trend based on percentage change
				if trendPercentage > 5 {
					overallTrend = "increasing"
				} else if trendPercentage < -5 {
					overallTrend = "decreasing"
				} else {
					overallTrend = "stable"
				}
			} else {
				overallTrend = "insufficient data"
				trendPercentage = 0
			}

			result := PipelineTrends{
				Buckets:         nonEmptyBuckets,
				OverallTrend:    overallTrend,
				TrendPercentage: trendPercentage,
				TotalPipelines:  len(pipelinesWithDuration),
				TimePeriodDays:  days,
				BucketSizeDays:  bucketSize,
				Branch:          branch,
			}

			return f.FormatAndPrint(result, format, jsonFlag)
		},
	}

	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Filter by branch or tag")
	cmd.Flags().IntVarP(&days, "days", "d", 30, "Number of days to analyze")
	cmd.Flags().IntVar(&bucketSize, "bucket-size", 7, "Size of time buckets in days (default: 7 for weekly buckets)")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}
