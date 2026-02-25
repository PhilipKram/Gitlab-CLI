package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PhilipKram/gitlab-cli/internal/api"
	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/PhilipKram/gitlab-cli/internal/config"
	"github.com/spf13/cobra"
)

// NewAPICmd creates the api command.
func NewAPICmd(f *cmdutil.Factory) *cobra.Command {
	var (
		method       string
		body         string
		headers      []string
		hostname     string
		fields       []string
		methodSet    bool
	)

	cmd := &cobra.Command{
		Use:   "api <endpoint>",
		Short: "Make authenticated API requests",
		Long: `Make authenticated requests to the GitLab API.

The endpoint can be a path like "projects" which will be resolved to the full API URL.
Or it can be a full URL starting with "http".`,
		Example: `  $ glab api projects
  $ glab api projects/:id/merge_requests
  $ glab api users --method GET
  $ glab api projects/:id/issues --method POST --body '{"title":"Bug"}'
  $ glab api projects/:id/issues -X POST -f title=Bug -f description="Fix it"
  $ glab api projects/:id/merge_requests/1/notes -f body="Looks good!"
  $ glab api graphql --method POST --body '{"query":"{ currentUser { name } }"}'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			methodSet = cmd.Flags().Changed("method")
			endpoint := args[0]

			// Build JSON body from --field flags (validate early before auth)
			if len(fields) > 0 {
				jsonObj := make(map[string]interface{})

				// If --body was also provided, use it as the base
				if body != "" {
					if err := json.Unmarshal([]byte(body), &jsonObj); err != nil {
						return fmt.Errorf("parsing --body JSON: %w", err)
					}
				}

				// Overlay --field values
				for _, f := range fields {
					parts := strings.SplitN(f, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid field format %q, expected key=value", f)
					}
					jsonObj[parts[0]] = parts[1]
				}

				b, err := json.Marshal(jsonObj)
				if err != nil {
					return fmt.Errorf("encoding fields to JSON: %w", err)
				}
				body = string(b)

				// Default to POST when fields are provided and method wasn't explicitly set
				if !methodSet {
					method = "POST"
				}
			}

			// Resolve host: --hostname flag > factory client > default
			host := hostname
			if host == "" {
				client, err := f.Client()
				if err == nil {
					host = client.Host()
				} else {
					host = config.DefaultHost()
				}
			}

			token, _ := config.TokenForHost(host)
			if token == "" {
				return fmt.Errorf("not authenticated with %s; run 'glab auth login --hostname %s'", host, host)
			}

			authMethod := config.AuthMethodForHost(host)

			// Build the full URL
			var url string
			if strings.HasPrefix(endpoint, "http") {
				url = endpoint
			} else {
				baseURL := api.APIURL(host)
				endpoint = strings.TrimPrefix(endpoint, "/")
				url = baseURL + "/" + endpoint
			}

			// Create request
			var reqBody io.Reader
			if body != "" {
				reqBody = strings.NewReader(body)
			}

			req, err := http.NewRequest(strings.ToUpper(method), url, reqBody)
			if err != nil {
				return fmt.Errorf("creating request: %w", err)
			}

			if authMethod == "oauth" {
				req.Header.Set("Authorization", "Bearer "+token)
			} else {
				req.Header.Set("PRIVATE-TOKEN", token)
			}
			req.Header.Set("Content-Type", "application/json")

			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("making request: %w", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %w", err)
			}

			// Try to pretty-print JSON
			var prettyJSON interface{}
			if err := json.Unmarshal(respBody, &prettyJSON); err == nil {
				formatted, err := json.MarshalIndent(prettyJSON, "", "  ")
				if err == nil {
					fmt.Fprintln(f.IOStreams.Out, string(formatted))
					return nil
				}
			}

			// Fall back to raw output
			fmt.Fprintln(f.IOStreams.Out, string(respBody))
			return nil
		},
	}

	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringVar(&body, "body", "", "Request body (JSON)")
	cmd.Flags().StringArrayVarP(&fields, "field", "f", nil, `Add a string field in "key=value" format`)
	cmd.Flags().StringSliceVarP(&headers, "header", "H", nil, "Additional headers (key:value)")
	cmd.Flags().StringVar(&hostname, "hostname", "", "GitLab hostname to use")

	return cmd
}
