package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterVariableTools registers all variable tools on the server.
func RegisterVariableTools(server *mcp.Server, f *cmdutil.Factory) {
	registerVariableList(server, f)
	registerVariableGet(server, f)
	registerVariableSet(server, f)
	registerVariableDelete(server, f)
}

func registerVariableList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_list",
		Description: "List CI/CD variables for a GitLab project",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		variables, _, err := client.ProjectVariables.ListVariables(project, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("listing variables: %w", err)
		}
		return textResult(variables)
	})
}

func registerVariableGet(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Key  string `json:"key"            jsonschema:"variable key"`
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_get",
		Description: "Get a specific CI/CD variable by key",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Key, "key"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		variable, _, err := client.ProjectVariables.GetVariable(project, in.Key, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("getting variable: %w", err)
		}
		return textResult(variable)
	})
}

func registerVariableSet(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Key       string `json:"key"                  jsonschema:"variable key"`
		Value     string `json:"value"                jsonschema:"variable value"`
		Repo      string `json:"repo,omitempty"       jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Masked    bool   `json:"masked,omitempty"     jsonschema:"mask variable value in logs"`
		Protected bool   `json:"protected,omitempty"  jsonschema:"only available in protected branches/tags"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_set",
		Description: "Create or update a CI/CD variable (upsert)",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Key, "key"); err != nil {
			return nil, nil, err
		}
		if err := requireString(in.Value, "value"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}

		scope := "*"
		varType := gitlab.EnvVariableType

		// Try update first
		updateOpts := &gitlab.UpdateProjectVariableOptions{
			Value:            &in.Value,
			Protected:        &in.Protected,
			Masked:           &in.Masked,
			EnvironmentScope: &scope,
			VariableType:     &varType,
		}
		_, _, err = client.ProjectVariables.UpdateVariable(project, in.Key, updateOpts)
		if err == nil {
			return plainResult(fmt.Sprintf("Updated variable %q", in.Key)), nil, nil
		}

		// Create if update fails
		createOpts := &gitlab.CreateProjectVariableOptions{
			Key:              &in.Key,
			Value:            &in.Value,
			Protected:        &in.Protected,
			Masked:           &in.Masked,
			EnvironmentScope: &scope,
			VariableType:     &varType,
		}
		_, _, err = client.ProjectVariables.CreateVariable(project, createOpts)
		if err != nil {
			return nil, nil, fmt.Errorf("creating variable: %w", err)
		}
		return plainResult(fmt.Sprintf("Created variable %q", in.Key)), nil, nil
	})
}

func registerVariableDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Key  string `json:"key"            jsonschema:"variable key"`
		Repo string `json:"repo,omitempty" jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_delete",
		Description: "Delete a CI/CD variable",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Key, "key"); err != nil {
			return nil, nil, err
		}
		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.ProjectVariables.RemoveVariable(project, in.Key, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting variable: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted variable %q", in.Key)), nil, nil
	})
}
