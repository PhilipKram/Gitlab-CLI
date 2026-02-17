package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/browser"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/tableprinter"
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

			release, _, err := client.Releases.CreateRelease(project, opts)
			if err != nil {
				return fmt.Errorf("creating release: %w", err)
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Created release %s\n", release.TagName)

			remote, _ := f.Remote()
			host := "gitlab.com"
			if remote != nil {
				host = remote.Host
			}
			releaseURL := api.WebURL(host, project+"/-/releases/"+release.TagName)
			fmt.Fprintln(out, releaseURL)

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
		jsonFlag bool
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

			releases, _, err := client.Releases.ListReleases(project, opts)
			if err != nil {
				return fmt.Errorf("listing releases: %w", err)
			}

			if len(releases) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No releases found")
				return nil
			}

			if jsonFlag {
				data, err := json.MarshalIndent(releases, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			tp := tableprinter.New(f.IOStreams.Out)
			for _, r := range releases {
				name := r.Name
				if name == "" {
					name = r.TagName
				}
				tp.AddRow(
					r.TagName,
					name,
					timeAgo(r.CreatedAt),
				)
			}
			return tp.Render()
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "L", 30, "Maximum number of results")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

	return cmd
}

func newReleaseViewCmd(f *cmdutil.Factory) *cobra.Command {
	var web bool
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
			release, _, err := client.Releases.GetRelease(project, tag)
			if err != nil {
				return fmt.Errorf("getting release: %w", err)
			}

			if web {
				remote, _ := f.Remote()
				host := "gitlab.com"
				if remote != nil {
					host = remote.Host
				}
				return browser.Open(api.WebURL(host, project+"/-/releases/"+tag))
			}

			if jsonFlag {
				data, err := json.MarshalIndent(release, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(f.IOStreams.Out, string(data))
				return nil
			}

			out := f.IOStreams.Out
			fmt.Fprintf(out, "Tag:     %s\n", release.TagName)
			fmt.Fprintf(out, "Name:    %s\n", release.Name)
			fmt.Fprintf(out, "Author:  %s\n", release.Author.Username)
			fmt.Fprintf(out, "Created: %s\n", timeAgo(release.CreatedAt))

			if len(release.Assets.Links) > 0 {
				fmt.Fprintln(out, "\nAssets:")
				for _, link := range release.Assets.Links {
					fmt.Fprintf(out, "  - %s: %s\n", link.Name, link.URL)
				}
			}

			if release.Description != "" {
				fmt.Fprintf(out, "\n%s\n", release.Description)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&web, "web", "w", false, "Open in browser")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")

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

			_, _, err = client.Releases.DeleteRelease(project, args[0])
			if err != nil {
				return fmt.Errorf("deleting release: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Deleted release %s\n", args[0])
			return nil
		},
	}

	return cmd
}

func newReleaseDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download <tag>",
		Short: "Download release assets",
		Long:  "List downloadable assets for a release.",
		Example: `  $ glab release download v1.0.0`,
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

			release, _, err := client.Releases.GetRelease(project, args[0])
			if err != nil {
				return fmt.Errorf("getting release: %w", err)
			}

			out := f.IOStreams.Out
			if len(release.Assets.Sources) > 0 {
				fmt.Fprintln(out, "Source archives:")
				for _, s := range release.Assets.Sources {
					fmt.Fprintf(out, "  %s: %s\n", s.Format, s.URL)
				}
			}

			if len(release.Assets.Links) > 0 {
				fmt.Fprintln(out, "\nAsset links:")
				for _, link := range release.Assets.Links {
					fmt.Fprintf(out, "  %s: %s\n", link.Name, link.URL)
				}
			}

			if len(release.Assets.Sources) == 0 && len(release.Assets.Links) == 0 {
				fmt.Fprintln(f.IOStreams.ErrOut, "No downloadable assets found")
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
		Use:   "upload <tag> <file>",
		Short: "Upload an asset to a release",
		Example: `  $ glab release upload v1.0.0 ./build/app.tar.gz --name "Application binary"`,
		Args: cobra.ExactArgs(2),
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

			link, _, err := client.ReleaseLinks.CreateReleaseLink(project, tag, linkOpts)
			if err != nil {
				return fmt.Errorf("creating release link: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Uploaded %s to release %s\n", link.Name, tag)
			fmt.Fprintf(f.IOStreams.Out, "%s\n", link.DirectAssetURL)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Display name for the asset")
	cmd.Flags().StringVar(&linkType, "type", "", "Link type: other, runbook, image, package")

	return cmd
}
