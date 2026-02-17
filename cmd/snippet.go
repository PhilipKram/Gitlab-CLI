package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewSnippetCmd creates the snippet command group.
func NewSnippetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snippet <command>",
		Short:   "Manage snippets",
		Long:    "Create, view, and manage GitLab snippets (similar to GitHub gists).",
		Aliases: []string{"snip"},
	}

	cmd.AddCommand(newSnippetCreateCmd(f))
	cmd.AddCommand(newSnippetListCmd(f))
	cmd.AddCommand(newSnippetViewCmd(f))
	cmd.AddCommand(newSnippetDeleteCmd(f))

	return cmd
}

func newSnippetCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		title      string
		filename   string
		visibility string
		filePath   string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a snippet",
		Example: `  $ glab snippet create --title "My snippet" --filename main.go --file ./main.go
  $ echo "content" | glab snippet create --title "From stdin" --filename snippet.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			var content string
			if filePath != "" {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("reading file: %w", err)
				}
				content = string(data)
			} else {
				return fmt.Errorf("--file flag is required")
			}

			if filename == "" {
				filename = filePath
			}

			vis := gitlab.PrivateVisibility
			switch visibility {
			case "public":
				vis = gitlab.PublicVisibility
			case "internal":
				vis = gitlab.InternalVisibility
			}

			opts := &gitlab.CreateSnippetOptions{
				Title:      &title,
				Visibility: &vis,
				Files: &[]*gitlab.CreateSnippetFileOptions{
					{
						FilePath: &filename,
						Content:  &content,
					},
				},
			}

			snippet, resp, err := client.Snippets.CreateSnippet(opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/snippets"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create snippet", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Created snippet #%d\n", snippet.ID)
			_, _ = fmt.Fprintf(f.IOStreams.Out, "%s\n", snippet.WebURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Snippet title (required)")
	cmd.Flags().StringVarP(&filename, "filename", "f", "", "Filename for the snippet content")
	cmd.Flags().StringVar(&visibility, "visibility", "private", "Visibility: public, internal, private")
	cmd.Flags().StringVar(&filePath, "file", "", "Path to file to use as snippet content")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newSnippetListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		stream   bool
		web      bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List snippets",
		Aliases: []string{"ls"},
		Example: `  $ glab snippet list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(api.WebURL(host, "/-/snippets"))
			}

			opts := &gitlab.ListSnippetsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
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
				fetchFunc := func(page int) ([]*gitlab.Snippet, *gitlab.Response, error) {
					pageOpts := *opts
					pageOpts.Page = int64(page)
					if pageOpts.PerPage == 0 {
						pageOpts.PerPage = 100
					}
					return client.Snippets.ListSnippets(&pageOpts)
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

				return cmdutil.FormatAndStream(f, results, outputFormat, limit, "snippets")
			}

			// Non-streaming mode: fetch all at once
			snippets, resp, err := client.Snippets.ListSnippets(opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/snippets"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list snippets", err)
			}

			if len(snippets) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No snippets found. Try increasing --limit.")
				return nil
			}

			return f.FormatAndPrint(snippets, format, jsonFlag)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming mode")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open snippets in browser")

	return cmd
}

func newSnippetViewCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		raw bool
		web bool
	)

	cmd := &cobra.Command{
		Use:   "view [<id>]",
		Short: "View a snippet",
		Example: `  $ glab snippet view 123
  $ glab snippet view 123 --raw`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return fmt.Errorf("snippet ID required")
			}

			id := strings.TrimPrefix(args[0], "#")
			snippetID, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid snippet ID: %s", args[0])
			}

			snippet, resp, err := client.Snippets.GetSnippet(snippetID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/snippets/" + id
				return errors.NewAPIError("GET", url, statusCode, "Failed to get snippet", err)
			}

			if web {
				return browser.Open(snippet.WebURL)
			}

			out := f.IOStreams.Out

			if raw {
				content, resp, err := client.Snippets.SnippetContent(snippetID)
				if err != nil {
					statusCode := 0
					if resp != nil {
						statusCode = resp.StatusCode
					}
					url := api.APIURL(client.Host()) + "/snippets/" + id + "/raw"
					return errors.NewAPIError("GET", url, statusCode, "Failed to get snippet content", err)
				}
				_, _ = fmt.Fprint(out, string(content))
				return nil
			}

			_, _ = fmt.Fprintf(out, "#%d %s\n", snippet.ID, snippet.Title)
			_, _ = fmt.Fprintf(out, "Visibility: %s\n", snippet.Visibility)
			_, _ = fmt.Fprintf(out, "Author:     %s\n", snippet.Author.Username)
			_, _ = fmt.Fprintf(out, "Created:    %s\n", timeAgo(snippet.CreatedAt))
			_, _ = fmt.Fprintf(out, "URL:        %s\n", snippet.WebURL)
			if len(snippet.Files) > 0 {
				_, _ = fmt.Fprintln(out, "\nFiles:")
				for _, file := range snippet.Files {
					_, _ = fmt.Fprintf(out, "  - %s\n", file.Path)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Output raw snippet content")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open snippet in browser")

	return cmd
}

func newSnippetDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete [<id>]",
		Short:   "Delete a snippet",
		Example: `  $ glab snippet delete 123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			if len(args) == 0 {
				return fmt.Errorf("snippet ID required")
			}

			id := strings.TrimPrefix(args[0], "#")
			snippetID, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid snippet ID: %s", args[0])
			}

			resp, err := client.Snippets.DeleteSnippet(snippetID)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/snippets/" + id
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete snippet", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted snippet #%d\n", snippetID)
			return nil
		},
	}

	return cmd
}
