package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
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

	return cmd
}

func newPipelineListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		status   string
		ref      string
		limit    int
		jsonFlag bool
		web      bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List pipelines",
		Aliases: []string{"ls"},
		Example: `  $ glab pipeline list
  $ glab pipeline list --status success --ref main
  $ glab pipeline list --limit 50`,
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

			pipelines, _, err := client.Pipelines.ListProjectPipelines(project, opts)
			if err != nil {
				return fmt.Errorf("listing pipelines: %w", err)
			}

			if len(pipelines) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No pipelines found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(pipelines, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, p := range pipelines {
				tp.AddRow(
					fmt.Sprintf("#%d", p.ID),
					p.Status,
					p.Ref,
					p.SHA[:8],
					pipelineTimeAgo(p.CreatedAt),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().StringVar(&status, "status", "", "Filter by status: running, pending, success, failed, canceled, skipped")
	cmd.Flags().StringVar(&ref, "ref", "", "Filter by branch or tag")
	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")

	return cmd
}

func newPipelineViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
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

			pipeline, _, err := client.Pipelines.GetPipeline(project, pipelineID)
			if err != nil {
				return fmt.Errorf("getting pipeline: %w", err)
			}

			if web {
				return browser.Open(pipeline.WebURL)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(pipeline, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Pipeline #%d\n", pipeline.ID)
			fmt.Fprintf(out, "Status:   %s\n", pipeline.Status)
			fmt.Fprintf(out, "Ref:      %s\n", pipeline.Ref)
			fmt.Fprintf(out, "SHA:      %s\n", pipeline.SHA)
			fmt.Fprintf(out, "Source:   %s\n", pipeline.Source)
			if pipeline.User != nil {
				fmt.Fprintf(out, "User:     %s\n", pipeline.User.Username)
			}
			fmt.Fprintf(out, "Created:  %s\n", pipelineTimeAgo(pipeline.CreatedAt))
			if pipeline.StartedAt != nil {
				fmt.Fprintf(out, "Started:  %s\n", pipelineTimeAgo(pipeline.StartedAt))
			}
			if pipeline.FinishedAt != nil {
				fmt.Fprintf(out, "Finished: %s\n", pipelineTimeAgo(pipeline.FinishedAt))
			}
			fmt.Fprintf(out, "Duration: %ds\n", pipeline.Duration)
			fmt.Fprintf(out, "URL:      %s\n", pipeline.WebURL)

			// Show jobs
			jobs, _, err := client.Jobs.ListPipelineJobs(project, pipelineID, nil)
			if err == nil && len(jobs) > 0 {
				fmt.Fprintln(out, "\nJobs:")
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
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newPipelineRunCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		ref       string
		variables []string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a new pipeline",
		Aliases: []string{"create", "trigger"},
		Example: `  $ glab pipeline run --ref main
  $ glab pipeline run --ref develop --variables KEY1=value1,KEY2=value2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.CreatePipelineOptions{
				Ref: &ref,
			}

			if len(variables) > 0 {
				var vars []*gitlab.PipelineVariableOptions
				for _, v := range variables {
					parts := strings.SplitN(v, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid variable format: %s (use KEY=value)", v)
					}
					varType := gitlab.VariableTypeValue("env_var")
					vars = append(vars, &gitlab.PipelineVariableOptions{
						Key:          &parts[0],
						Value:        &parts[1],
						VariableType: &varType,
					})
				}
				opts.Variables = &vars
			}

			pipeline, _, err := client.Pipelines.CreatePipeline(project, opts)
			if err != nil {
				return fmt.Errorf("creating pipeline: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Created pipeline #%d\n", pipeline.ID)
			fmt.Fprintf(out, "Status: %s\n", pipeline.Status)
			fmt.Fprintf(out, "%s\n", pipeline.WebURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&ref, "ref", "b", "", "Branch or tag to run pipeline on (required)")
	cmd.Flags().StringSliceVar(&variables, "variables", nil, "Pipeline variables (KEY=value)")
	_ = cmd.MarkFlagRequired("ref")

	return cmd
}

func newPipelineCancelCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel [<id>]",
		Short: "Cancel a running pipeline",
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

			pipeline, _, err := client.Pipelines.CancelPipelineBuild(project, pipelineID)
			if err != nil {
				return fmt.Errorf("canceling pipeline: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Canceled pipeline #%d (status: %s)\n", pipeline.ID, pipeline.Status)
			return nil
		},
	}

	return cmd
}

func newPipelineRetryCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry [<id>]",
		Short: "Retry a failed pipeline",
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

			pipeline, _, err := client.Pipelines.RetryPipelineBuild(project, pipelineID)
			if err != nil {
				return fmt.Errorf("retrying pipeline: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Retried pipeline #%d (status: %s)\n", pipeline.ID, pipeline.Status)
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

			_, err = client.Pipelines.DeletePipeline(project, pipelineID)
			if err != nil {
				return fmt.Errorf("deleting pipeline: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted pipeline #%d\n", pipelineID)
			return nil
		},
	}

	return cmd
}

func newPipelineJobsCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "jobs [<pipeline-id>]",
		Short: "List jobs in a pipeline",
		Example: `  $ glab pipeline jobs 12345`,
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

			jobs, _, err := client.Jobs.ListPipelineJobs(project, pipelineID, nil)
			if err != nil {
				return fmt.Errorf("listing jobs: %w", err)
			}

			if len(jobs) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No jobs found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(jobs, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
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

			reader, _, err := client.Jobs.GetTraceFile(project, jobID)
			if err != nil {
				return fmt.Errorf("getting job trace: %w", err)
			}

			buf := make([]byte, 4096)
			for {
				n, readErr := reader.Read(buf)
				if n > 0 {
					fmt.Fprint(f.IOStreams.Out, string(buf[:n]))
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
				fmt.Fprint(f.IOStreams.Out, string(buf[:n]))
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
		Use:   "retry-job [<job-id>]",
		Short: "Retry a specific failed job",
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
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			fmt.Fprintf(f.IOStreams.Out, "Retried job #%d (status: %s)\n", job.ID, job.Status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newPipelineCancelJobCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "cancel-job [<job-id>]",
		Short: "Cancel a specific running job",
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
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			fmt.Fprintf(f.IOStreams.Out, "Canceled job #%d (status: %s)\n", job.ID, job.Status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newPipelineArtifactsCmd(f *cmdutil.Factory) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "artifacts [<job-id>]",
		Short: "Download job artifacts as a zip file",
		Example: `  $ glab pipeline artifacts 67890
  $ glab pipeline artifacts 67890 --output my-artifacts.zip`,
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

			// Use default output path if not specified
			if outputPath == "" {
				outputPath = "artifacts.zip"
			}

			reader, _, err := client.Jobs.GetJobArtifacts(project, jobID)
			if err != nil {
				return fmt.Errorf("downloading job artifacts: %w", err)
			}

			// Create output file
			outFile, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("creating output file: %w", err)
			}
			defer outFile.Close()

			// Copy artifacts to file
			written, err := io.Copy(outFile, reader)
			if err != nil {
				return fmt.Errorf("writing artifacts to file: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Downloaded artifacts to %s (%d bytes)\n", outputPath, written)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: artifacts.zip)")

	return cmd
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
