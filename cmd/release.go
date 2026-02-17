package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/errors"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// NewReleaseCmd creates the release command group.
func NewReleaseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release <command>",
		Short: "Manage releases",
		Long:  "Create, view, and manage GitLab releases.",
	}

	cmd.AddCommand(newReleaseCreateCmd(f))
	cmd.AddCommand(newReleaseListCmd(f))
	cmd.AddCommand(newReleaseViewCmd(f))
	cmd.AddCommand(newReleaseDeleteCmd(f))
	cmd.AddCommand(newReleaseDownloadCmd(f))
	cmd.AddCommand(newReleaseUploadCmd(f))

	return cmd
}

func newReleaseCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name        string
		description string
		ref         string
		milestones  []string
		assets      []string
		web         bool
	)

	cmd := &cobra.Command{
		Use:   "create <tag>",
		Short: "Create a release",
		Example: `  $ glab release create v1.0.0 --name "Version 1.0" --description "First release"
  $ glab release create v2.0.0 --ref main --name "Version 2.0"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			tag := args[0]
			opts := &gitlab.CreateReleaseOptions{
				TagName:     &tag,
				Name:        &name,
				Description: &description,
			}

			if ref != "" {
				opts.Ref = &ref
			}

			if len(milestones) > 0 {
				opts.Milestones = &milestones
			}

			if len(assets) > 0 {
				var links []*gitlab.ReleaseAssetLinkOptions
				for _, a := range assets {
					linkName := a
					linkURL := a
					links = append(links, &gitlab.ReleaseAssetLinkOptions{
						Name: &linkName,
						URL:  &linkURL,
					})
				}
				opts.Assets = &gitlab.ReleaseAssetsOptions{
					Links: links,
				}
			}

			release, resp, err := client.Releases.CreateRelease(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/releases"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create release", err)
			}

			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "Created release %s\n", release.TagName)

			remote, _ := f.Remote()
			host := "gitlab.com"
			if remote != nil {
				host = remote.Host
			}
			releaseURL := api.WebURL(host, project+"/-/releases/"+release.TagName)
			_, _ = fmt.Fprintln(out, releaseURL)

			if web {
				_ = browser.Open(releaseURL)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Release name")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Release description")
	cmd.Flags().StringVar(&ref, "ref", "", "Branch or commit SHA (creates tag from this ref)")
	cmd.Flags().StringSliceVar(&milestones, "milestone", nil, "Associated milestones")
	cmd.Flags().StringSliceVar(&assets, "asset", nil, "Release asset URLs")
	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser after creation")

	return cmd
}

func newReleaseListCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		limit    int
		format   string
		jsonFlag bool
		stream   bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List releases",
		Aliases: []string{"ls"},
		Example: `  $ glab release list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			opts := &gitlab.ListReleasesOptions{
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
				fetchFunc := func(page int) ([]*gitlab.Release, *gitlab.Response, error) {
					pageOpts := *opts
					pageOpts.Page = int64(page)
					if pageOpts.PerPage == 0 {
						pageOpts.PerPage = 100
					}
					return client.Releases.ListReleases(project, &pageOpts)
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

				return cmdutil.FormatAndStream(f, results, outputFormat, limit, "releases")
			}

			// Non-streaming mode: fetch all at once
			releases, resp, err := client.Releases.ListReleases(project, opts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/releases"
				return errors.NewAPIError("GET", url, statusCode, "Failed to list releases", err)
			}

			if len(releases) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No releases found. Try increasing --limit or check that releases exist for this project.")
				return nil
			}

			return f.FormatAndPrint(releases, string(outputFormat), false)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")
	cmd.Flags().BoolVar(&stream, "stream", false, "Enable streaming mode")

	return cmd
}

func newReleaseViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
	var format string
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "view <tag>",
		Short: "View a release",
		Example: `  $ glab release view v1.0.0
  $ glab release view v1.0.0 --web`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			tag := args[0]
			release, resp, err := client.Releases.GetRelease(project, tag)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/releases/" + tag
				return errors.NewAPIError("GET", url, statusCode, "Failed to get release", err)
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(api.WebURL(host, project+"/-/releases/"+tag))
			}

			// Backward compatibility: --json flag sets format to json
			if jsonFlag {
				format = "json"
			}

			// Use formatter for non-default formats
			if format != "" && format != "table" {
				return f.FormatAndPrint(release, format, false)
			}

			// Default custom display
			out := f.IOStreams.Out
			_, _ = fmt.Fprintf(out, "Tag:     %s\n", release.TagName)
			_, _ = fmt.Fprintf(out, "Name:    %s\n", release.Name)
			_, _ = fmt.Fprintf(out, "Author:  %s\n", release.Author.Username)
			_, _ = fmt.Fprintf(out, "Created: %s\n", timeAgo(release.CreatedAt))

			if len(release.Assets.Links) > 0 {
				_, _ = fmt.Fprintln(out, "\nAssets:")
				for _, link := range release.Assets.Links {
					_, _ = fmt.Fprintf(out, "  - %s: %s\n", link.Name, link.URL)
				}
			}

			if release.Description != "" {
				_, _ = fmt.Fprintf(out, "\n%s\n", release.Description)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().StringVarP(&format, "format", "F", "table", "Output format: json, table, or plain")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON (deprecated: use --format=json)")

	return cmd
}

func newReleaseDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <tag>",
		Short:   "Delete a release",
		Example: `  $ glab release delete v1.0.0`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			_, resp, err := client.Releases.DeleteRelease(project, args[0])
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/releases/" + args[0]
				return errors.NewAPIError("DELETE", url, statusCode, "Failed to delete release", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Deleted release %s\n", args[0])
			return nil
		},
	}

	return cmd
}

func newReleaseDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download <tag>",
		Short:   "Download release assets",
		Long:    "List downloadable assets for a release.",
		Example: `  $ glab release download v1.0.0`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			release, resp, err := client.Releases.GetRelease(project, args[0])
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/releases/" + args[0]
				return errors.NewAPIError("GET", url, statusCode, "Failed to get release", err)
			}

			out := f.IOStreams.Out
			if len(release.Assets.Sources) > 0 {
				_, _ = fmt.Fprintln(out, "Source archives:")
				for _, s := range release.Assets.Sources {
					_, _ = fmt.Fprintf(out, "  %s: %s\n", s.Format, s.URL)
				}
			}

			if len(release.Assets.Links) > 0 {
				_, _ = fmt.Fprintln(out, "\nAsset links:")
				for _, link := range release.Assets.Links {
					_, _ = fmt.Fprintf(out, "  %s: %s\n", link.Name, link.URL)
				}
			}

			if len(release.Assets.Sources) == 0 && len(release.Assets.Links) == 0 {
				_, _ = fmt.Fprintln(f.IOStreams.ErrOut, "No downloadable assets found")
			}

			return nil
		},
	}

	return cmd
}

func newReleaseUploadCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		name     string
		linkType string
	)

	cmd := &cobra.Command{
		Use:     "upload <tag> <file>",
		Short:   "Upload an asset to a release",
		Example: `  $ glab release upload v1.0.0 ./build/app.tar.gz --name "Application binary"`,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := f.Client()
			if err != nil {
				return err
			}

			project, err := f.FullProjectPath()
			if err != nil {
				return err
			}

			tag := args[0]
			filePath := args[1]

			// Verify file exists
			if _, err := os.Stat(filePath); err != nil {
				return fmt.Errorf("file not found: %w", err)
			}

			// Create a release link (user must host the file externally or use GitLab package registry)
			if name == "" {
				name = filePath
			}

			// For direct asset links, user should provide a URL; for local files, use a placeholder
			fileURL := filePath

			lt := gitlab.OtherLinkType
			if linkType != "" {
				lt = gitlab.LinkTypeValue(linkType)
			}

			linkOpts := &gitlab.CreateReleaseLinkOptions{
				Name:     &name,
				URL:      &fileURL,
				LinkType: &lt,
			}

			link, resp, err := client.ReleaseLinks.CreateReleaseLink(project, tag, linkOpts)
			if err != nil {
				statusCode := 0
				if resp != nil {
					statusCode = resp.StatusCode
				}
				url := api.APIURL(client.Host()) + "/projects/" + project + "/releases/" + tag + "/assets/links"
				return errors.NewAPIError("POST", url, statusCode, "Failed to create release link", err)
			}

			_, _ = fmt.Fprintf(f.IOStreams.Out, "Uploaded %s to release %s\n", link.Name, tag)
			_, _ = fmt.Fprintf(f.IOStreams.Out, "%s\n", link.DirectAssetURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Display name for the asset")
	cmd.Flags().StringVar(&linkType, "type", "", "Link type: other, runbook, image, package")

	return cmd
}
