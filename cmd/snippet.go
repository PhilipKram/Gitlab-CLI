package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewSnippetCmd creates the snippet command group.
func NewSnippetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snippet <command>",
		Short: "Manage snippets",
		Long:  "Create, view, and manage GitLab snippets (similar to GitHub gists).",
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

			snippet, _, err := client.Snippets.CreateSnippet(opts)
			if err != nil {
				return fmt.Errorf("creating snippet: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Created snippet #%d\n", snippet.ID)
			fmt.Fprintf(f.IOStreams.Out, "%s\n", snippet.WebURL)
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
		jsonFlag bool
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

			opts := &gitlab.ListSnippetsOptions{
				ListOptions: gitlab.ListOptions{PerPage: int64(limit)},
			}

			snippets, _, err := client.Snippets.ListSnippets(opts)
			if err != nil {
				return fmt.Errorf("listing snippets: %w", err)
			}

			if len(snippets) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No snippets found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(snippets, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, s := range snippets {
				tp.AddRow(
					fmt.Sprintf("#%d", s.ID),
					s.Title,
					string(s.Visibility),
					timeAgo(s.CreatedAt),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newSnippetViewCmd(f *cmdutil.Factory) *cobra.Command {
	var raw bool

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

			snippet, _, err := client.Snippets.GetSnippet(snippetID)
			if err != nil {
				return fmt.Errorf("getting snippet: %w", err)
			}

			out := f.IOStreams.Out

			if raw {
				content, _, err := client.Snippets.SnippetContent(snippetID)
				if err != nil {
					return fmt.Errorf("getting snippet content: %w", err)
				}
				fmt.Fprint(out, string(content))
				return nil
			}

			fmt.Fprintf(out, "#%d %s\n", snippet.ID, snippet.Title)
			fmt.Fprintf(out, "Visibility: %s\n", snippet.Visibility)
			fmt.Fprintf(out, "Author:     %s\n", snippet.Author.Username)
			fmt.Fprintf(out, "Created:    %s\n", timeAgo(snippet.CreatedAt))
			fmt.Fprintf(out, "URL:        %s\n", snippet.WebURL)
			if len(snippet.Files) > 0 {
				fmt.Fprintln(out, "\nFiles:")
				for _, file := range snippet.Files {
					fmt.Fprintf(out, "  - %s\n", file.Path)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Output raw snippet content")

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

			_, err = client.Snippets.DeleteSnippet(snippetID)
			if err != nil {
				return fmt.Errorf("deleting snippet: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted snippet #%d\n", snippetID)
			return nil
		},
	}

	return cmd
}
