package tools

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterPipelineAnalyticsTools registers pipeline analytics tools on the server.
func RegisterPipelineAnalyticsTools(server *mcp.Server, f *cmdutil.Factory) {
	registerPipelineStats(server, f)
	registerPipelineTrends(server, f)
	registerPipelineSlowestJobs(server, f)
	registerPipelineFlaky(server, f)
}

// fetchAllPipelines fetches all pipelines for a project within the given time period.
func fetchAllPipelines(client *api.Client, project string, days int64, branch string) ([]*gitlab.PipelineInfo, error) {
	cutoffDate := time.Now().AddDate(0, 0, -int(days))
	opts := &gitlab.ListProjectPipelinesOptions{
		ListOptions:  gitlab.ListOptions{PerPage: 100},
		UpdatedAfter: &cutoffDate,
	}
	if branch != "" {
		opts.Ref = &branch
	}

	var all []*gitlab.PipelineInfo
	page := 1
	for {
		opts.Page = int64(page)
		pipelines, resp, err := client.Pipelines.ListProjectPipelines(project, opts)
		if err != nil {
			return nil, fmt.Errorf("listing pipelines: %w", err)
		}
		if len(pipelines) == 0 {
			break
		}
		all = append(all, pipelines...)
		if resp.NextPage == 0 {
			break
		}
		page = int(resp.NextPage)
	}
	return all, nil
}

func registerPipelineStats(server *mcp.Server, f *cmdutil.Factory) {
	type pipelineStats struct {
		TotalPipelines int     `json:"total_pipelines"`
		SuccessCount   int     `json:"success_count"`
		FailedCount    int     `json:"failed_count"`
		CanceledCount  int     `json:"canceled_count"`
		SkippedCount   int     `json:"skipped_count"`
		RunningCount   int     `json:"running_count"`
		PendingCount   int     `json:"pending_count"`
		SuccessRate    float64 `json:"success_rate"`
		FailureRate    float64 `json:"failure_rate"`
		TimePeriodDays int64   `json:"time_period_days"`
		Branch         string  `json:"branch,omitempty"`
	}

	type Input struct {
		Repo   string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Branch string `json:"branch,omitempty" jsonschema:"filter by branch or tag"`
		Days   int64  `json:"days,omitempty"   jsonschema:"number of days to analyze (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_stats",
		Description: "Show pipeline success/failure rate statistics",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		days := in.Days
		if days <= 0 {
			days = 30
		}

		pipelines, err := fetchAllPipelines(client, project, days, in.Branch)
		if err != nil {
			return nil, nil, err
		}

		stats := pipelineStats{
			TotalPipelines: len(pipelines),
			TimePeriodDays: days,
			Branch:         in.Branch,
		}
		for _, p := range pipelines {
			switch p.Status {
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
		if stats.TotalPipelines > 0 {
			stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalPipelines) * 100
			stats.FailureRate = float64(stats.FailedCount) / float64(stats.TotalPipelines) * 100
		}
		return textResult(stats)
	})
}

func registerPipelineTrends(server *mcp.Server, f *cmdutil.Factory) {
	type trendBucket struct {
		StartDate       time.Time `json:"start_date"`
		EndDate         time.Time `json:"end_date"`
		PipelineCount   int       `json:"pipeline_count"`
		AverageDuration float64   `json:"average_duration"`
		MinDuration     float64   `json:"min_duration"`
		MaxDuration     float64   `json:"max_duration"`
		TotalDuration   float64   `json:"total_duration"`
	}

	type pipelineTrends struct {
		Buckets         []trendBucket `json:"buckets"`
		OverallTrend    string        `json:"overall_trend"`
		TrendPercentage float64       `json:"trend_percentage"`
		TotalPipelines  int           `json:"total_pipelines"`
		TimePeriodDays  int64         `json:"time_period_days"`
		BucketSizeDays  int64         `json:"bucket_size_days"`
		Branch          string        `json:"branch,omitempty"`
	}

	type Input struct {
		Repo       string `json:"repo,omitempty"        jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Branch     string `json:"branch,omitempty"      jsonschema:"filter by branch or tag"`
		Days       int64  `json:"days,omitempty"        jsonschema:"number of days to analyze (default 30)"`
		BucketSize int64  `json:"bucket_size,omitempty" jsonschema:"size of time buckets in days (default 7)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_trends",
		Description: "Analyze pipeline duration trends over time",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		days := in.Days
		if days <= 0 {
			days = 30
		}
		bucketSize := in.BucketSize
		if bucketSize <= 0 {
			bucketSize = 7
		}

		pipelineInfos, err := fetchAllPipelines(client, project, days, in.Branch)
		if err != nil {
			return nil, nil, err
		}
		if len(pipelineInfos) == 0 {
			return plainResult("No pipelines found in the specified time period"), nil, nil
		}

		// Fetch full pipeline details for duration
		var pipelines []*gitlab.Pipeline
		for _, info := range pipelineInfos {
			p, _, err := client.Pipelines.GetPipeline(project, info.ID)
			if err != nil {
				continue
			}
			pipelines = append(pipelines, p)
		}
		if len(pipelines) == 0 {
			return plainResult("No pipeline details could be fetched"), nil, nil
		}

		// Create time buckets
		numBuckets := (days + bucketSize - 1) / bucketSize
		buckets := make([]trendBucket, numBuckets)
		now := time.Now()
		for i := int64(0); i < numBuckets; i++ {
			buckets[i] = trendBucket{
				StartDate: now.AddDate(0, 0, -int(days)+(int(i)*int(bucketSize))),
				EndDate:   now.AddDate(0, 0, -int(days)+((int(i)+1)*int(bucketSize))),
			}
		}

		for _, p := range pipelines {
			if p.CreatedAt == nil || p.Duration == 0 {
				continue
			}
			for i := range buckets {
				if (p.CreatedAt.Equal(buckets[i].StartDate) || p.CreatedAt.After(buckets[i].StartDate)) &&
					p.CreatedAt.Before(buckets[i].EndDate) {
					buckets[i].PipelineCount++
					buckets[i].TotalDuration += float64(p.Duration)
					if buckets[i].MinDuration == 0 || float64(p.Duration) < buckets[i].MinDuration {
						buckets[i].MinDuration = float64(p.Duration)
					}
					if float64(p.Duration) > buckets[i].MaxDuration {
						buckets[i].MaxDuration = float64(p.Duration)
					}
					break
				}
			}
		}

		var nonEmpty []trendBucket
		for _, b := range buckets {
			if b.PipelineCount > 0 {
				b.AverageDuration = b.TotalDuration / float64(b.PipelineCount)
				nonEmpty = append(nonEmpty, b)
			}
		}
		if len(nonEmpty) == 0 {
			return plainResult("No completed pipelines with duration found"), nil, nil
		}

		sort.Slice(nonEmpty, func(i, j int) bool {
			return nonEmpty[i].StartDate.Before(nonEmpty[j].StartDate)
		})

		var overallTrend string
		var trendPct float64
		if len(nonEmpty) >= 2 {
			first := nonEmpty[0].AverageDuration
			last := nonEmpty[len(nonEmpty)-1].AverageDuration
			if first > 0 {
				trendPct = ((last - first) / first) * 100
			}
			if trendPct > 5 {
				overallTrend = "increasing"
			} else if trendPct < -5 {
				overallTrend = "decreasing"
			} else {
				overallTrend = "stable"
			}
		} else {
			overallTrend = "insufficient data"
		}

		result := pipelineTrends{
			Buckets:         nonEmpty,
			OverallTrend:    overallTrend,
			TrendPercentage: trendPct,
			TotalPipelines:  len(pipelines),
			TimePeriodDays:  days,
			BucketSizeDays:  bucketSize,
			Branch:          in.Branch,
		}
		return textResult(result)
	})
}

func registerPipelineSlowestJobs(server *mcp.Server, f *cmdutil.Factory) {
	type jobDurationStats struct {
		JobName         string  `json:"job_name"`
		Stage           string  `json:"stage"`
		TotalRuns       int     `json:"total_runs"`
		AverageDuration float64 `json:"average_duration"`
		MinDuration     float64 `json:"min_duration"`
		MaxDuration     float64 `json:"max_duration"`
		TotalDuration   float64 `json:"total_duration"`
	}

	type slowestJobsResult struct {
		Jobs           []jobDurationStats `json:"jobs"`
		TotalPipelines int                `json:"total_pipelines"`
		TotalJobs      int                `json:"total_jobs"`
		TimePeriodDays int64              `json:"time_period_days"`
		Branch         string             `json:"branch,omitempty"`
	}

	type Input struct {
		Repo   string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Branch string `json:"branch,omitempty" jsonschema:"filter by branch or tag"`
		Days   int64  `json:"days,omitempty"   jsonschema:"number of days to analyze (default 30)"`
		Limit  int64  `json:"limit,omitempty"  jsonschema:"maximum number of jobs to return (default 10)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_slowest_jobs",
		Description: "Identify slowest CI jobs by average duration",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		days := in.Days
		if days <= 0 {
			days = 30
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}

		pipelineInfos, err := fetchAllPipelines(client, project, days, in.Branch)
		if err != nil {
			return nil, nil, err
		}
		if len(pipelineInfos) == 0 {
			return plainResult("No pipelines found in the specified time period"), nil, nil
		}

		type jobAgg struct {
			name      string
			stage     string
			durations []float64
		}
		jobStats := make(map[string]*jobAgg)
		totalJobs := 0

		for _, pi := range pipelineInfos {
			jobs, _, err := client.Jobs.ListPipelineJobs(project, pi.ID, nil)
			if err != nil {
				continue
			}
			for _, job := range jobs {
				totalJobs++
				if _, ok := jobStats[job.Name]; !ok {
					jobStats[job.Name] = &jobAgg{name: job.Name, stage: job.Stage}
				}
				jobStats[job.Name].durations = append(jobStats[job.Name].durations, job.Duration)
			}
		}

		var stats []jobDurationStats
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
			stats = append(stats, jobDurationStats{
				JobName:         agg.name,
				Stage:           agg.stage,
				TotalRuns:       len(agg.durations),
				AverageDuration: total / float64(len(agg.durations)),
				MinDuration:     min,
				MaxDuration:     max,
				TotalDuration:   total,
			})
		}

		sort.Slice(stats, func(i, j int) bool {
			return stats[i].AverageDuration > stats[j].AverageDuration
		})
		if int64(len(stats)) > limit {
			stats = stats[:limit]
		}

		return textResult(slowestJobsResult{
			Jobs:           stats,
			TotalPipelines: len(pipelineInfos),
			TotalJobs:      totalJobs,
			TimePeriodDays: days,
			Branch:         in.Branch,
		})
	})
}

func registerPipelineFlaky(server *mcp.Server, f *cmdutil.Factory) {
	type flakyJobStats struct {
		JobName       string  `json:"job_name"`
		Stage         string  `json:"stage"`
		TotalRuns     int     `json:"total_runs"`
		SuccessCount  int     `json:"success_count"`
		FailureCount  int     `json:"failure_count"`
		FlakinessRate float64 `json:"flakiness_rate"`
		LastFailure   string  `json:"last_failure,omitempty"`
		LastSuccess   string  `json:"last_success,omitempty"`
	}

	type flakyJobsResult struct {
		Jobs           []flakyJobStats `json:"jobs"`
		TotalPipelines int             `json:"total_pipelines"`
		TotalJobs      int             `json:"total_jobs"`
		FlakyJobs      int             `json:"flaky_jobs"`
		TimePeriodDays int64           `json:"time_period_days"`
		Branch         string          `json:"branch,omitempty"`
	}

	type Input struct {
		Repo      string  `json:"repo,omitempty"      jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Branch    string  `json:"branch,omitempty"    jsonschema:"filter by branch or tag"`
		Days      int64   `json:"days,omitempty"      jsonschema:"number of days to analyze (default 30)"`
		Threshold float64 `json:"threshold,omitempty" jsonschema:"minimum flakiness rate threshold 0-100 (default 0 shows all)"`
		Limit     int64   `json:"limit,omitempty"     jsonschema:"maximum number of flaky jobs to return (default 20)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_flaky",
		Description: "Detect flaky jobs with inconsistent pass/fail patterns",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		days := in.Days
		if days <= 0 {
			days = 30
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}

		pipelineInfos, err := fetchAllPipelines(client, project, days, in.Branch)
		if err != nil {
			return nil, nil, err
		}
		if len(pipelineInfos) == 0 {
			return plainResult("No pipelines found in the specified time period"), nil, nil
		}

		type flakyAgg struct {
			name         string
			stage        string
			successCount int
			failureCount int
			lastSuccess  string
			lastFailure  string
		}
		jobStats := make(map[string]*flakyAgg)
		totalJobs := 0

		for _, pi := range pipelineInfos {
			jobs, _, err := client.Jobs.ListPipelineJobs(project, pi.ID, nil)
			if err != nil {
				continue
			}
			for _, job := range jobs {
				totalJobs++
				if _, ok := jobStats[job.Name]; !ok {
					jobStats[job.Name] = &flakyAgg{name: job.Name, stage: job.Stage}
				}
				agg := jobStats[job.Name]
				switch job.Status {
				case "success":
					agg.successCount++
					if job.FinishedAt != nil {
						agg.lastSuccess = job.FinishedAt.Format(time.RFC3339)
					}
				case "failed":
					agg.failureCount++
					if job.FinishedAt != nil {
						agg.lastFailure = job.FinishedAt.Format(time.RFC3339)
					}
				}
			}
		}

		var stats []flakyJobStats
		for _, agg := range jobStats {
			total := agg.successCount + agg.failureCount
			if total == 0 || agg.successCount == 0 || agg.failureCount == 0 {
				continue
			}
			rate := float64(agg.failureCount) / float64(total) * 100
			if in.Threshold > 0 && rate < in.Threshold {
				continue
			}
			stats = append(stats, flakyJobStats{
				JobName:       agg.name,
				Stage:         agg.stage,
				TotalRuns:     total,
				SuccessCount:  agg.successCount,
				FailureCount:  agg.failureCount,
				FlakinessRate: rate,
				LastFailure:   agg.lastFailure,
				LastSuccess:   agg.lastSuccess,
			})
		}

		// Sort by distance from 50% (most flaky first)
		sort.Slice(stats, func(i, j int) bool {
			distI := stats[i].FlakinessRate - 50.0
			if distI < 0 {
				distI = -distI
			}
			distJ := stats[j].FlakinessRate - 50.0
			if distJ < 0 {
				distJ = -distJ
			}
			return distI < distJ
		})
		if int64(len(stats)) > limit {
			stats = stats[:limit]
		}

		return textResult(flakyJobsResult{
			Jobs:           stats,
			TotalPipelines: len(pipelineInfos),
			TotalJobs:      totalJobs,
			FlakyJobs:      len(stats),
			TimePeriodDays: days,
			Branch:         in.Branch,
		})
	})
}
