package tools

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// RegisterVariableTools registers all CI/CD variable tools on the server.
func RegisterVariableTools(server *mcp.Server, f *cmdutil.Factory) {
	registerVariableList(server, f)
	registerVariableGet(server, f)
	registerVariableSet(server, f)
	registerVariableDelete(server, f)
}

func registerVariableList(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group string `json:"group,omitempty" jsonschema:"list group-level variables (specify group path)"`
		Limit int64  `json:"limit,omitempty" jsonschema:"maximum number of results (default 30)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_list",
		Description: "List CI/CD variables for a project or group",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			variables, _, err := client.GroupVariables.ListVariables(in.Group, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("listing group variables: %w", err)
			}
			return textResult(variables)
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		variables, _, err := client.ProjectVariables.ListVariables(project, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("listing project variables: %w", err)
		}
		return textResult(variables)
	})
}

func registerVariableGet(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Key   string `json:"key"             jsonschema:"variable key"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group string `json:"group,omitempty" jsonschema:"get group-level variable (specify group path)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_get",
		Description: "Get a CI/CD variable by key",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Key, "key"); err != nil {
			return nil, nil, err
		}

		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			variable, _, err := client.GroupVariables.GetVariable(in.Group, in.Key, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("getting group variable: %w", err)
			}
			return textResult(variable)
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		variable, _, err := client.ProjectVariables.GetVariable(project, in.Key, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("getting project variable: %w", err)
		}
		return textResult(variable)
	})
}

func registerVariableSet(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Key       string `json:"key"                    jsonschema:"variable key"`
		Value     string `json:"value"                  jsonschema:"variable value"`
		Repo      string `json:"repo,omitempty"         jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group     string `json:"group,omitempty"        jsonschema:"set group-level variable (specify group path)"`
		Masked    bool   `json:"masked,omitempty"       jsonschema:"mask variable value in logs"`
		Protected bool   `json:"protected,omitempty"    jsonschema:"protect variable (only available in protected branches/tags)"`
		Scope     string `json:"scope,omitempty"        jsonschema:"environment scope (default *)"`
		Type      string `json:"type,omitempty"         jsonschema:"variable type: env_var or file (default env_var)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_set",
		Description: "Create or update a CI/CD variable",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Key, "key"); err != nil {
			return nil, nil, err
		}
		if err := requireString(in.Value, "value"); err != nil {
			return nil, nil, err
		}

		scope := in.Scope
		if scope == "" {
			scope = "*"
		}
		variableType := gitlab.EnvVariableType
		if in.Type == "file" {
			variableType = gitlab.FileVariableType
		}

		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			// Try update first, then create
			updateOpts := &gitlab.UpdateGroupVariableOptions{
				Value:            &in.Value,
				Protected:        &in.Protected,
				Masked:           &in.Masked,
				EnvironmentScope: &scope,
				VariableType:     &variableType,
			}
			_, _, err = client.GroupVariables.UpdateVariable(in.Group, in.Key, updateOpts)
			if err != nil {
				createOpts := &gitlab.CreateGroupVariableOptions{
					Key:              &in.Key,
					Value:            &in.Value,
					Protected:        &in.Protected,
					Masked:           &in.Masked,
					EnvironmentScope: &scope,
					VariableType:     &variableType,
				}
				_, _, err = client.GroupVariables.CreateVariable(in.Group, createOpts)
				if err != nil {
					return nil, nil, fmt.Errorf("setting group variable: %w", err)
				}
				return plainResult(fmt.Sprintf("Created group variable %q", in.Key)), nil, nil
			}
			return plainResult(fmt.Sprintf("Updated group variable %q", in.Key)), nil, nil
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		updateOpts := &gitlab.UpdateProjectVariableOptions{
			Value:            &in.Value,
			Protected:        &in.Protected,
			Masked:           &in.Masked,
			EnvironmentScope: &scope,
			VariableType:     &variableType,
		}
		_, _, err = client.ProjectVariables.UpdateVariable(project, in.Key, updateOpts)
		if err != nil {
			createOpts := &gitlab.CreateProjectVariableOptions{
				Key:              &in.Key,
				Value:            &in.Value,
				Protected:        &in.Protected,
				Masked:           &in.Masked,
				EnvironmentScope: &scope,
				VariableType:     &variableType,
			}
			_, _, err = client.ProjectVariables.CreateVariable(project, createOpts)
			if err != nil {
				return nil, nil, fmt.Errorf("setting project variable: %w", err)
			}
			return plainResult(fmt.Sprintf("Created variable %q", in.Key)), nil, nil
		}
		return plainResult(fmt.Sprintf("Updated variable %q", in.Key)), nil, nil
	})
}

func registerVariableDelete(server *mcp.Server, f *cmdutil.Factory) {
	type Input struct {
		Key   string `json:"key"             jsonschema:"variable key to delete"`
		Repo  string `json:"repo,omitempty"  jsonschema:"repository in OWNER/REPO or HOST/OWNER/REPO format"`
		Group string `json:"group,omitempty" jsonschema:"delete group-level variable (specify group path)"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "variable_delete",
		Description: "Delete a CI/CD variable",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in Input) (*mcp.CallToolResult, any, error) {
		if err := requireString(in.Key, "key"); err != nil {
			return nil, nil, err
		}

		if in.Group != "" {
			client, err := f.Client()
			if err != nil {
				return nil, nil, err
			}
			_, err = client.GroupVariables.RemoveVariable(in.Group, in.Key, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("deleting group variable: %w", err)
			}
			return plainResult(fmt.Sprintf("Deleted group variable %q", in.Key)), nil, nil
		}

		client, project, err := resolveClientAndProject(f, in.Repo)
		if err != nil {
			return nil, nil, err
		}
		_, err = client.ProjectVariables.RemoveVariable(project, in.Key, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("deleting project variable: %w", err)
		}
		return plainResult(fmt.Sprintf("Deleted variable %q", in.Key)), nil, nil
	})
}
