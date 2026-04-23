package cmd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewPipelineCmd creates the pipeline command group.
func NewPipelineCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pipeline <command>",
		Short:   "Manage pipelines and CI/CD",
		Long:    "View, run, and manage GitLab CI/CD pipelines.",
		Aliases: []string{"ci", "pipe"},
	}

	cmd.AddCommand(newPipelineListCmd(f))
	cmd.AddCommand(newPipelineViewCmd(f))
	cmd.AddCommand(newPipelineRunCmd(f))
	cmd.AddCommand(newPipelineCancelCmd(f))
	cmd.AddCommand(newPipelineRetryCmd(f))
	cmd.AddCommand(newPipelineDeleteCmd(f))
	cmd.AddCommand(newPipelineJobsCmd(f))
	cmd.AddCommand(newPipelineJobLogCmd(f))
	cmd.AddCommand(newPipelineRetryJobCmd(f))
	cmd.AddCommand(newPipelineCancelJobCmd(f))
	cmd.AddCommand(newPipelineArtifactsCmd(f))
	cmd.AddCommand(newPipelineStatsCmd(f))
	cmd.AddCommand(newPipelineSlowestJobsCmd(f))
	cmd.AddCommand(newPipelineTrendsCmd(f))
	cmd.AddCommand(newPipelineFlakyCmd(f))
	cmd.AddCommand(newPipelineWatchCmd(f))
	cmd.AddCommand(newCILintCmd(f))

	return cmd
}

func newPipelineListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		status   string
		ref      string
		limit    int
		format   string
		jsonFlag bool
		web      bool
		stream   bool
		orderBy  string
		sort     string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List pipelines",
		Aliases: []string{"ls"},
		Example: `  $ glab pipeline list
  $ glab pipeline list --status success --ref main
  $ glab pipeline list --limit 50
  $ glab pipeline list --order-by id --sort desc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(fmt.Sprintf("https://%s/%s/-/pipelines", host, project))
			}

			opts := &gitlab.ListProjectPipelinesOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			if status != "" {
				pipelineStatus := gitlab.BuildStateValue(status)
				opts.Status = &pipelineStatus
			}
			if ref != "" {
				opts.Ref = &ref
			}
			if orderBy != "" {
				opts.OrderBy = &orderBy
			}
			if sort != "" {
				opts.Sort = &sort
			}

			outputFormat, err := f.ResolveFormat(format, jsonFlag)
			if err != nil {
				return err
			}

			// Use streaming mode if --stream flag is set
			if stream {
				// Create context for pagination
				ctx := context.Background()

				// Create fetch function for pagination
				fetchFunc := func(page int) ([]*gitlab.PipelineInfo, *gitlab.Response, error) {
					pageOpts := *opts
					pageOpts.Page = int64(page)
					if pageOpts.PerPage == 0 {
						pageOpts.PerPage = 100
					}
					return client.Pipelines.ListProjectPipelines(project, &pageOpts)
				}

				// Configure pagination options
				paginateOpts := api.PaginateOptions{
					PerPage:    int(opts.PerPage),
					BufferSize: 100,
				}
				if limit > 0 && limit < 100 {
					paginateOpts.PerPage = limit
					paginateOpts.BufferSize = limit
				}

				// Start pagination
				results := api.PaginateToChannel(ctx, fetchFunc, paginateOpts)

				return cmdutil.FormatAndStream(f, results, outputFormat, limit, "pipelines")
			}

			// Non-streaming mode: fetch all at once
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
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No pipelines found. Try adjusting filters or increase --limit.")
				return nil
			}

			return f.FormatAndPrint(pipelines, format, jsonFlag)
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status: running, pending, success, failed, canceled, skipped")
	cmd.Flags().StringVar(&ref, "ref", "", "Filter by branch or tag")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming mode")
	cmd.Flags().StringVar(&orderBy, "order-by", "", "Order by: id, status, ref, updated_at, user_id")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order: asc or desc")

	return cmd
}

func newPipelineViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var format string
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View a pipeline",
		Example: `  $ glab pipeline view 12345
  $ glab pipeline view 12345 --web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			pipelineID, err := parsePipelineArg(args)
			if err != nil {
				return err
			}

			pipeline, resp, err := client.Pipelines.GetPipeline(project, pipelineID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/pipelines/" + strconv.FormatInt(pipelineID, 10)
				return errors.NewAPIError("GET", url, statusCode, "Failed to get pipeline", err)
			}

			if web {
				return browser.Open(pipeline.WebURL)
			}

			// If non-default format requested, use formatter
			if jsonFlag || (format != "" && format != "table") {
				return f.FormatAndPrint(pipeline, format, jsonFlag)
			}

			// Default custom display
			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "Pipeline #%d\n", pipeline.ID)
			_, _ = fmt.Fprintf(out, "Status:   %s\n", pipeline.Status)
			_, _ = fmt.Fprintf(out, "Ref:      %s\n", pipeline.Ref)
			_, _ = fmt.Fprintf(out, "SHA:      %s\n", pipeline.SHA)
			_, _ = fmt.Fprintf(out, "Source:   %s\n", pipeline.Source)
			if pipeline.User != nil {
				_, _ = fmt.Fprintf(out, "User:     %s\n", pipeline.User.Username)
			}
			_, _ = fmt.Fprintf(out, "Created:  %s\n", pipelineTimeAgo(pipeline.CreatedAt))
			if pipeline.StartedAt != nil {
				_, _ = fmt.Fprintf(out, "Started:  %s\n", pipelineTimeAgo(pipeline.StartedAt))
			}
			if pipeline.FinishedAt != nil {
				_, _ = fmt.Fprintf(out, "Finished: %s\n", pipelineTimeAgo(pipeline.FinishedAt))
			}
			_, _ = fmt.Fprintf(out, "Duration: %ds\n", pipeline.Duration)
			_, _ = fmt.Fprintf(out, "URL:      %s\n", pipeline.WebURL)

			// Show jobs
			jobs, _, err := client.Jobs.ListPipelineJobs(project, pipelineID, nil)
			if err == nil && len(jobs) > 0 {
				_, _ = fmt.Fprintln(out, "\nJobs:")
				tp := tableprinter.New(out)
				for _, j := range jobs {
					tp.AddRow(
						fmt.Sprintf("  %d", j.ID),
						j.Name,
						j.Stage,
						j.Status,
						fmt.Sprintf("%ds", int(j.Duration)),
					)
				}
				_ = tp.Render()
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newPipelineRunCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		ref           string
		branch        string
		variables     []string
		cancelRunning bool
	)

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a new pipeline",
		Aliases: []string{"create", "trigger"},
		Example: `  $ glab pipeline run --branch main
  $ glab pipeline run --ref develop --variables KEY1=value1,KEY2=value2
  $ glab pipeline run --ref feature/my-branch --variables "HOTFIX_IMAGES=a,b,c"
  $ glab pipeline run --ref main --cancel-running`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --branch is an alias for --ref
			if branch != "" && ref == "" {
				ref = branch
			}
			if ref == "" {
				return fmt.Errorf("required flag \"ref\" (or \"branch\") not set")
			}

			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			// Parse variables
			varsMap := make(map[string]string)
			for _, v := range variables {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid variable format: %s (use KEY=value)", v)
				}
				varsMap[parts[0]] = parts[1]
			}

			out := f.IOStreams.Out

			// Cancel running/pending pipelines on the same ref if requested
			if cancelRunning {
				for _, pipelineStatus := range []string{"running", "pending"} {
					s := gitlab.BuildStateValue(pipelineStatus)
					listOpts := &gitlab.ListProjectPipelinesOptions{
						Status: &s,
						Ref:    &ref,
					}
					pipelines, _, err := client.Pipelines.ListProjectPipelines(project, listOpts)
					if err != nil {
						return fmt.Errorf("listing %s pipelines: %w", pipelineStatus, err)
					}
					for _, p := range pipelines {
						_, _, cancelErr := client.Pipelines.CancelPipelineBuild(project, int64(p.ID))
						if cancelErr != nil {
							_, _ = fmt.Fprintf(f.IOStreams.ErrOut, "Warning: failed to cancel pipeline #%d: %v\n", p.ID, cancelErr)
							continue
						}
						_, _ = fmt.Fprintf(out, "Canceled pipeline #%d\n", p.ID)
					}
				}
			}

			pipeline, err := runPipelineWithTrigger(client, project, ref, varsMap)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(out, "Created pipeline #%d\n", pipeline.ID)
			_, _ = fmt.Fprintf(out, "Status: %s\n", pipeline.Status)
			_, _ = fmt.Fprintf(out, "%s\n", pipeline.WebURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&ref, "ref", "b", "", "Branch or tag to run pipeline on (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "Alias for --ref")
	cmd.Flags().Lookup("branch").Hidden = true
	cmd.Flags().StringArrayVar(&variables, "variables", nil, "Pipeline variables (KEY=value)")
	cmd.Flags().BoolVar(&cancelRunning, "cancel-running", false, "Cancel running/pending pipelines on the same ref before triggering")

	return cmd
}

// getOrCreateTriggerToken returns an existing pipeline trigger token for the project,
// or creates one if none exist.
func getOrCreateTriggerToken(client *api.Client, project string) (string, error) {
	triggers, _, err := client.PipelineTriggers.ListPipelineTriggers(project, nil)
	if err != nil {
		return "", fmt.Errorf("listing trigger tokens: %w", err)
	}

	for _, t := range triggers {
		if t.Token != "" {
			return t.Token, nil
		}
	}

	// No trigger tokens exist — create one
	desc := "glab-cli"
	newTrigger, _, err := client.PipelineTriggers.AddPipelineTrigger(project, &gitlab.AddPipelineTriggerOptions{
		Description: &desc,
	})
	if err != nil {
		return "", fmt.Errorf("creating trigger token: %w", err)
	}
	return newTrigger.Token, nil
}

// runPipelineWithTrigger runs a pipeline using the trigger API.
// It auto-detects or creates a trigger token for the project.
func runPipelineWithTrigger(client *api.Client, project, ref string, variables map[string]string) (*gitlab.Pipeline, error) {
	token, err := getOrCreateTriggerToken(client, project)
	if err != nil {
		return nil, err
	}

	opts := &gitlab.RunPipelineTriggerOptions{
		Ref:   &ref,
		Token: &token,
	}
	if len(variables) > 0 {
		opts.Variables = variables
	}

	pipeline, resp, err := client.PipelineTriggers.RunPipelineTrigger(project, opts)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		url := api.APIURL(client.Host()) + "/projects/" + project + "/trigger/pipeline"
		return nil, errors.NewAPIError("POST", url, statusCode, "Failed to trigger pipeline", err)
	}
	return pipeline, nil
}

func newPipelineCancelCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cancel [<id>]",
		Short:   "Cancel a running pipeline",
		Example: `  $ glab pipeline cancel 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			pipelineID, err := parsePipelineArg(args)
			if err != nil {
				return err
			}

			pipeline, resp, err := client.Pipelines.CancelPipelineBuild(project, pipelineID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/pipelines/" + strconv.FormatInt(pipelineID, 10) + "/cancel"
				return errors.NewAPIError("POST", url, statusCode, "Failed to cancel pipeline", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Canceled pipeline #%d (status: %s)\n", pipeline.ID, pipeline.Status)
			return nil
		},
	}

	return cmd
}

func newPipelineRetryCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "retry [<id>]",
		Short:   "Retry a failed pipeline",
		Example: `  $ glab pipeline retry 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			pipelineID, err := parsePipelineArg(args)
			if err != nil {
				return err
			}

			pipeline, resp, err := client.Pipelines.RetryPipelineBuild(project, pipelineID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/pipelines/" + strconv.FormatInt(pipelineID, 10) + "/retry"
				return errors.NewAPIError("POST", url, statusCode, "Failed to retry pipeline", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Retried pipeline #%d (status: %s)\n", pipeline.ID, pipeline.Status)
			return nil
		},
	}

	return cmd
}

func newPipelineDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [<id>]",
		Short:   "Delete a pipeline",
		Example: `  $ glab pipeline delete 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			pipelineID, err := parsePipelineArg(args)
			if err != nil {
				return err
			}

			resp, err := client.Pipelines.DeletePipeline(project, pipelineID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/pipelines/" + strconv.FormatInt(pipelineID, 10)
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete pipeline", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted pipeline #%d\n", pipelineID)
			return nil
		},
	}

	return cmd
}

func newPipelineJobsCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		jsonFlag    bool
		limit       int
		statusScope string
	)

	cmd := &cobra.Command{
		Use:     "jobs [<pipeline-id>]",
		Short:   "List jobs in a pipeline",
		Example: `  $ glab pipeline jobs 12345
  $ glab pipeline jobs 12345 --status running --limit 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			pipelineID, err := parsePipelineArg(args)
			if err != nil {
				return err
			}

			opts := &gitlab.ListJobsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}
			if statusScope != "" {
				scope := []gitlab.BuildStateValue{gitlab.BuildStateValue(statusScope)}
				opts.Scope = &scope
			}

			jobs, resp, err := client.Jobs.ListPipelineJobs(project, pipelineID, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/pipelines/" + strconv.FormatInt(pipelineID, 10) + "/jobs"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list pipeline jobs", err)
			}

			if len(jobs) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No jobs found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(jobs, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, j := range jobs {
				tp.AddRow(
					fmt.Sprintf("%d", j.ID),
					j.Name,
					j.Stage,
					j.Status,
					fmt.Sprintf("%.0fs", j.Duration),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().IntVarP(&limit, "limit", "L", 100, "Maximum number of results")
	cmd.Flags().StringVar(&statusScope, "status", "", "Filter by status: running, success, failed, pending, canceled, skipped, manual")

	return cmd
}

func newPipelineJobLogCmd(f *cmdutil.Factory) *cobra.Command {
	var follow bool

	cmd := &cobra.Command{
		Use:     "job-log [<job-id>]",
		Short:   "View the log/trace of a job",
		Aliases: []string{"trace"},
		Example: `  $ glab pipeline job-log 67890
  $ glab pipeline job-log 67890 --follow`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return fmt.Errorf("job ID required")
			}

			jobID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid job ID: %s", args[0])
			}

			if follow {
				return followJobLog(f, client, project, int(jobID))
			}

			reader, resp, err := client.Jobs.GetTraceFile(project, jobID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/jobs/" + strconv.FormatInt(jobID, 10) + "/trace"
				return errors.NewAPIError("GET", url, statusCode, "Failed to get job trace", err)
			}

			buf := make([]byte, 4096)
			for {
				n, readErr := reader.Read(buf)
				if n > 0 {
					_, _ = fmt.Fprint(f.IOStreams.Out, string(buf[:n]))
				}
				if readErr != nil {
					break
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream job log in real-time")

	return cmd
}

func followJobLog(f *cmdutil.Factory, client *api.Client, project string, jobID int) error {
	var lastBytePos int64
	jobIDInt64 := int64(jobID)

	for {
		// Get job status to check if still running
		job, _, err := client.Jobs.GetJob(project, jobIDInt64)
		if err != nil {
			return fmt.Errorf("getting job status: %w", err)
		}

		// Fetch trace from last position
		reader, _, err := client.Jobs.GetTraceFile(project, jobIDInt64)
		if err != nil {
			return fmt.Errorf("getting job trace: %w", err)
		}

		// Skip to last position
		if lastBytePos > 0 {
			buf := make([]byte, lastBytePos)
			_, _ = reader.Read(buf)
		}

		// Read and print new content
		buf := make([]byte, 4096)
		for {
			n, readErr := reader.Read(buf)
			if n > 0 {
				_, _ = fmt.Fprint(f.IOStreams.Out, string(buf[:n]))
				lastBytePos += int64(n)
			}
			if readErr != nil {
				break
			}
		}

		// Check if job is finished
		jobFinished := job.Status == "success" || job.Status == "failed" ||
			job.Status == "canceled" || job.Status == "skipped"

		if jobFinished {
			break
		}

		// Wait before next poll
		time.Sleep(2 * time.Second)
	}

	return nil
}

func newPipelineRetryJobCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:     "retry-job [<job-id>]",
		Short:   "Retry a specific failed job",
		Example: `  $ glab pipeline retry-job 67890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return fmt.Errorf("job ID required")
			}

			jobID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid job ID: %s", args[0])
			}

			job, _, err := client.Jobs.RetryJob(project, jobID)
			if err != nil {
				return fmt.Errorf("retrying job: %w", err)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(job, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Retried job #%d (status: %s)\n", job.ID, job.Status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newPipelineCancelJobCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:     "cancel-job [<job-id>]",
		Short:   "Cancel a specific running job",
		Example: `  $ glab pipeline cancel-job 67890`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return fmt.Errorf("job ID required")
			}

			jobID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid job ID: %s", args[0])
			}

			job, _, err := client.Jobs.CancelJob(project, jobID)
			if err != nil {
				return fmt.Errorf("canceling job: %w", err)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(job, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Canceled job #%d (status: %s)\n", job.ID, job.Status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newPipelineArtifactsCmd(f *cmdutil.Factory) *cobra.Command {
	var outputPath string
	var filePath string

	cmd := &cobra.Command{
		Use:   "artifacts [<job-id>]",
		Short: "Download job artifacts as a zip file",
		Example: `  $ glab pipeline artifacts 67890
  $ glab pipeline artifacts 67890 --output my-artifacts.zip
  $ glab pipeline artifacts 67890 --path path/to/file.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return fmt.Errorf("job ID required")
			}

			jobID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid job ID: %s", args[0])
			}

			reader, _, err := client.Jobs.GetJobArtifacts(project, jobID)
			if err != nil {
				return fmt.Errorf("downloading job artifacts: %w", err)
			}

			// If --path is specified, extract only that file
			if filePath != "" {
				return extractFileFromArtifacts(f, reader, filePath, outputPath)
			}

			// Use default output path if not specified
			if outputPath == "" {
				outputPath = "artifacts.zip"
			}

			// Create output file
			outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer func() { _ = outFile.Close() }()

			// Copy artifacts to file
			written, err := io.Copy(outFile, reader)
			if err != nil {
				return fmt.Errorf("writing artifacts to file: %w", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Downloaded artifacts to %s (%d bytes)\n", outputPath, written)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: artifacts.zip)")
	cmd.Flags().StringVar(&filePath, "path", "", "Extract a specific file from artifacts")

	return cmd
}

func extractFileFromArtifacts(f *cmdutil.Factory, reader io.Reader, filePath string, outputPath string) error {
	// Create a temporary file to store the zip
	tmpFile, err := os.CreateTemp("", "glab-artifacts-*.zip")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Copy artifacts to temp file
	_, err = io.Copy(tmpFile, reader)
	if err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("writing artifacts to temporary file: %w", err)
	}
	_ = tmpFile.Close()

	// Open the zip file
	zipReader, err := zip.OpenReader(tmpPath)
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}
	defer func() { _ = zipReader.Close() }()

	// Find and extract the specified file
	for _, zipFile := range zipReader.File {
		if zipFile.Name == filePath {
			// Determine output path
			if outputPath == "" {
				outputPath = filepath.Base(filePath)
			}

			// Open the file in the zip
			rc, err := zipFile.Open()
			if err != nil {
				return fmt.Errorf("opening file in zip: %w", err)
			}
			defer func() { _ = rc.Close() }()

			// Create output file
			outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer func() { _ = outFile.Close() }()

			// Copy the file
			written, err := io.Copy(outFile, rc)
			if err != nil {
				return fmt.Errorf("extracting file: %w", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Extracted %s to %s (%d bytes)\n", filePath, outputPath, written)
			return nil
		}
	}

	return fmt.Errorf("file %s not found in artifacts", filePath)
}

func parsePipelineArg(args []string) (int64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("pipeline ID required")
	}
	id := strings.TrimPrefix(args[0], "#")
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid pipeline ID: %s", args[0])
	}
	return n, nil
}

func pipelineTimeAgo(t *time.Time) string {
	return timeAgo(t)
}
