package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterPipelineTools registers all pipeline tools on the server.
func RegisterPipelineTools(server *mcp.Server, f *cmdutil.Factory) {
	registerPipelineList(server, f)
	registerPipelineView(server, f)
	registerPipelineRun(server, f)
	registerPipelineCancel(server, f)
	registerPipelineRetry(server, f)
	registerPipelineDelete(server, f)
	registerPipelineJobs(server, f)
	registerPipelineJobLog(server, f)
}

func registerPipelineList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo   string `json:"repo,omitempty"   jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Status string `json:"status,omitempty" jsonschema:"filter by status: running, pending, success, failed, canceled, skipped, created, manual"`
		Branch string `json:"branch,omitempty" jsonschema:"filter by branch name"`
		Limit  int64  `json:"limit,omitempty"  jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_list",
		Description: "List CI/CD pipelines for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		opts := &gitlab.ListProjectPipelinesOptions{
			ListOptions: gitlab.ListOptions{PerPage: clampPerPage(in.Limit)},
		}
		if in.Status != "" {
			status := gitlab.BuildStateValue(in.Status)
			opts.Status = &status
		}
		if in.Branch != "" {
			opts.Ref = &in.Branch
		}
		pipelines, _, err := client.Pipelines.ListProjectPipelines(project, opts)
		if err != nil {
			return nil, nil, fmt.Errorf("listing pipelines: %w", err)
		}
		return textResult(pipelines)
	})
}

func registerPipelineView(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Pipeline int64  `json:"pipeline"        jsonschema:"pipeline ID"`
		Repo     string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_view",
		Description: "View details of a specific pipeline",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Pipeline, "pipeline"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		pipeline, _, err := client.Pipelines.GetPipeline(project, in.Pipeline)
		if err != nil {
			return nil, nil, fmt.Errorf("getting pipeline: %w", err)
		}
		return textResult(pipeline)
	})
}

func registerPipelineRun(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Ref          string `json:"ref"                    jsonschema:"branch or tag to run the pipeline on"`
		Repo         string `json:"repo,omitempty"         jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Variables    string `json:"variables,omitempty"     jsonschema:"pipeline variables as KEY=value pairs, comma-separated"`
		TriggerToken string `json:"trigger_token,omitempty" jsonschema:"pipeline trigger token; when set uses the trigger API which properly resolves the branch CI config"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_run",
		Description: "Trigger a new pipeline on a branch or tag. Use trigger_token for pipelines that need the branch's own CI config (e.g. feature branches with new jobs).",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Ref, "ref"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}

		// Parse variables
		varsMap := make(map[string]string)
		if in.Variables != "" {
			for _, pair := range strings.Split(in.Variables, ",") {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					return nil, nil, fmt.Errorf("invalid variable %q: use KEY=value", pair)
				}
				varsMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		var pipeline *gitlab.Pipeline
		if in.TriggerToken != "" {
			// Use the trigger API — properly resolves branch CI config
			opts := &gitlab.RunPipelineTriggerOptions{
				Ref:   &in.Ref,
				Token: &in.TriggerToken,
			}
			if len(varsMap) > 0 {
				opts.Variables = varsMap
			}
			pipeline, _, err = client.PipelineTriggers.RunPipelineTrigger(project, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("triggering pipeline: %w", err)
			}
		} else {
			// Use the standard pipelines API
			opts := &gitlab.CreatePipelineOptions{Ref: &in.Ref}
			if len(varsMap) > 0 {
				var vars []*gitlab.PipelineVariableOptions
				for k, v := range varsMap {
					key := k
					value := v
					varType := gitlab.VariableTypeValue("env_var")
					vars = append(vars, &gitlab.PipelineVariableOptions{
						Key:          &key,
						Value:        &value,
						VariableType: &varType,
					})
				}
				opts.Variables = &vars
			}
			pipeline, _, err = client.Pipelines.CreatePipeline(project, opts)
			if err != nil {
				return nil, nil, fmt.Errorf("creating pipeline: %w", err)
			}
		}

		return plainResult(fmt.Sprintf("Created pipeline #%d (status: %s)\n%s", pipeline.ID, pipeline.Status, pipeline.WebURL)), nil, nil
	})
}

func registerPipelineCancel(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Pipeline int64  `json:"pipeline"        jsonschema:"pipeline ID"`
		Repo     string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_cancel",
		Description: "Cancel a running pipeline",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Pipeline, "pipeline"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		pipeline, _, err := client.Pipelines.CancelPipelineBuild(project, in.Pipeline)
		if err != nil {
			return nil, nil, fmt.Errorf("canceling pipeline: %w", err)
		}
		return plainResult(fmt.Sprintf("Canceled pipeline #%d (status: %s)", pipeline.ID, pipeline.Status)), nil, nil
	})
}

func registerPipelineRetry(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Pipeline int64  `json:"pipeline"        jsonschema:"pipeline ID"`
		Repo     string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_retry",
		Description: "Retry a failed pipeline",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Pipeline, "pipeline"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		pipeline, _, err := client.Pipelines.RetryPipelineBuild(project, in.Pipeline)
		if err != nil {
			return nil, nil, fmt.Errorf("retrying pipeline: %w", err)
		}
		return plainResult(fmt.Sprintf("Retried pipeline #%d (status: %s)", pipeline.ID, pipeline.Status)), nil, nil
	})
}

func registerPipelineDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Pipeline int64  `json:"pipeline"        jsonschema:"pipeline ID"`
		Repo     string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_delete",
		Description: "Delete a pipeline",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Pipeline, "pipeline"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.Pipelines.DeletePipeline(project, in.Pipeline)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting pipeline: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted pipeline #%d", in.Pipeline)), nil, nil
	})
}

func registerPipelineJobs(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Pipeline int64  `json:"pipeline"        jsonschema:"pipeline ID"`
		Repo     string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_jobs",
		Description: "List jobs in a pipeline",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Pipeline, "pipeline"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		jobs, _, err := client.Jobs.ListPipelineJobs(project, in.Pipeline, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("listing jobs: %w", err)
		}
		return textResult(jobs)
	})
}

func registerPipelineJobLog(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Job  int64  `json:"job"             jsonschema:"job ID"`
		Repo string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "pipeline_job_log",
		Description: "Get the log output of a pipeline job",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireID(in.Job, "job"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		reader, _, err := client.Jobs.GetTraceFile(project, in.Job)
		if err != nil {
			return nil, nil, fmt.Errorf("getting job log: %w", err)
		}
		log, err := readLog(reader)
		if err != nil {
			return nil, nil, fmt.Errorf("reading job log: %w", err)
		}
		return plainResult(log), nil, nil
	})
}
