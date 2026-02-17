package prompts

import (
	"context"
	"fmt"

	"github.com/PhilipKram/gitlab-cli/internal/cmdutil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterPrompts registers all GitLab prompt templates on the server.
func RegisterPrompts(server *mcp.Server, f *cmdutil.Factory) {
	registerReviewMRPrompt(server, f)
	registerExplainPipelineFailurePrompt(server, f)
	registerSummarizeIssuesPrompt(server, f)
	registerDraftMRDescriptionPrompt(server, f)
	registerCreateReleaseNotesPrompt(server, f)
}

func registerReviewMRPrompt(server *mcp.Server, f *cmdutil.Factory) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "review_mr",
		Description: "Structured instructions for reviewing a GitLab merge request",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "repo",
				Description: "Repository in OWNER/REPO format (optional, uses current repo if not specified)",
				Required:    false,
			},
			{
				Name:        "mr_id",
				Description: "Merge request IID to review",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		mrID := req.Params.Arguments["mr_id"]
		if mrID == "" {
			return nil, fmt.Errorf("mr_id is required")
		}

		repo := req.Params.Arguments["repo"]
		repoInfo := ""
		if repo != "" {
			repoInfo = fmt.Sprintf(" in repository %s", repo)
		}

		promptText := fmt.Sprintf(`Please review merge request !%s%s with the following structured approach:

## Code Quality
- Review code for clarity, maintainability, and adherence to best practices
- Check for potential bugs, edge cases, or error handling issues
- Assess code complexity and suggest simplifications where appropriate
- Verify naming conventions and code organization

## Tests
- Verify that appropriate tests are included (unit, integration, etc.)
- Check test coverage for new and modified code
- Ensure tests are meaningful and test the right scenarios
- Validate that edge cases are tested

## Documentation
- Check if code changes require documentation updates
- Verify that complex logic has adequate inline comments
- Ensure public APIs have proper documentation
- Confirm that README or other docs are updated if needed

## Security
- Review for potential security vulnerabilities
- Check for sensitive data exposure (credentials, tokens, etc.)
- Verify input validation and sanitization
- Assess authentication and authorization logic if applicable

## Performance
- Identify potential performance bottlenecks
- Review database queries for efficiency (N+1 queries, missing indexes)
- Check for unnecessary computations or loops
- Assess memory usage patterns

## GitLab-Specific
- Review CI/CD pipeline changes if .gitlab-ci.yml is modified
- Check for breaking changes to the API or CLI
- Verify that MR description accurately reflects the changes

Please use the mr_view and mr_diff tools to examine the merge request, then provide a structured review following these guidelines.`, mrID, repoInfo)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Review guidance for merge request !%s%s", mrID, repoInfo),
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})
}

func registerExplainPipelineFailurePrompt(server *mcp.Server, f *cmdutil.Factory) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "explain_pipeline_failure",
		Description: "Structured instructions for diagnosing and explaining a GitLab pipeline failure",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "repo",
				Description: "Repository in OWNER/REPO format (optional, uses current repo if not specified)",
				Required:    false,
			},
			{
				Name:        "pipeline_id",
				Description: "Pipeline ID to analyze",
				Required:    true,
			},
			{
				Name:        "job_id",
				Description: "Specific job ID to focus on (optional, analyzes all failed jobs if not specified)",
				Required:    false,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		pipelineID := req.Params.Arguments["pipeline_id"]
		if pipelineID == "" {
			return nil, fmt.Errorf("pipeline_id is required")
		}

		repo := req.Params.Arguments["repo"]
		repoInfo := ""
		if repo != "" {
			repoInfo = fmt.Sprintf(" in repository %s", repo)
		}

		jobID := req.Params.Arguments["job_id"]
		jobInfo := ""
		if jobID != "" {
			jobInfo = fmt.Sprintf(" (specifically job #%s)", jobID)
		}

		promptText := fmt.Sprintf(`Please analyze pipeline #%s%s%s to diagnose the failure with the following structured approach:

## Initial Assessment
- Use pipeline_view to get the overall pipeline status and list of jobs
- Identify which job(s) failed and their failure modes
- Note the pipeline trigger (commit, merge request, scheduled, manual)
- Check the branch and commit that triggered the pipeline

## Log Analysis
- Use pipeline_job_log or read the job log resource to retrieve failure logs
- Identify the specific error messages or stack traces
- Look for patterns: compilation errors, test failures, timeout, infrastructure issues
- Note the timing: did it fail immediately or after running for a while?

## Root Cause Investigation
- **Build/Compilation Failures**: Check for syntax errors, missing dependencies, version mismatches
- **Test Failures**: Identify which tests failed and why (assertion errors, environment issues, flaky tests)
- **Infrastructure Issues**: Check for resource constraints (disk space, memory, timeouts), network issues, service dependencies
- **Configuration Problems**: Review .gitlab-ci.yml changes, environment variables, runner configuration
- **Dependency Issues**: Check for breaking changes in external dependencies, registry access issues

## Environment & Context
- Check if this is a new failure or regression (compare with previous successful pipelines)
- Review recent commits and changes that might have caused the failure
- Verify if the failure is reproducible or intermittent
- Check if similar jobs in the pipeline succeeded or failed

## Recommended Fixes
Based on the analysis, provide specific, actionable recommendations:
- Immediate fixes to resolve the failure
- Code changes needed (with examples if applicable)
- Configuration updates required
- Dependencies to update or pin
- Infrastructure or runner configuration changes
- Preventive measures to avoid similar failures

## Next Steps
- Suggest verification steps after implementing fixes
- Recommend additional checks or tests to add
- Propose improvements to CI/CD configuration if applicable

Please use the pipeline_view, pipeline_job_log, and related tools to gather information, then provide a structured diagnosis following these guidelines.`, pipelineID, repoInfo, jobInfo)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Diagnostic guidance for pipeline #%s%s%s", pipelineID, repoInfo, jobInfo),
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})
}

func registerSummarizeIssuesPrompt(server *mcp.Server, f *cmdutil.Factory) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "summarize_issues",
		Description: "Structured instructions for summarizing GitLab issues",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "repo",
				Description: "Repository in OWNER/REPO format (optional, uses current repo if not specified)",
				Required:    false,
			},
			{
				Name:        "state",
				Description: "Issue state to filter by: opened, closed, or all (optional, defaults to opened)",
				Required:    false,
			},
			{
				Name:        "labels",
				Description: "Comma-separated list of labels to filter by (optional)",
				Required:    false,
			},
			{
				Name:        "assignee",
				Description: "Username to filter issues by assignee (optional)",
				Required:    false,
			},
			{
				Name:        "milestone",
				Description: "Milestone title to filter by (optional)",
				Required:    false,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		repo := req.Params.Arguments["repo"]
		repoInfo := ""
		if repo != "" {
			repoInfo = fmt.Sprintf(" in repository %s", repo)
		}

		state := req.Params.Arguments["state"]
		if state == "" {
			state = "opened"
		}

		var filters []string
		if labels := req.Params.Arguments["labels"]; labels != "" {
			filters = append(filters, fmt.Sprintf("labels: %s", labels))
		}
		if assignee := req.Params.Arguments["assignee"]; assignee != "" {
			filters = append(filters, fmt.Sprintf("assignee: %s", assignee))
		}
		if milestone := req.Params.Arguments["milestone"]; milestone != "" {
			filters = append(filters, fmt.Sprintf("milestone: %s", milestone))
		}

		filterInfo := ""
		if len(filters) > 0 {
			filterInfo = fmt.Sprintf(" (filtered by %s)", fmt.Sprintf("%v", filters))
		}

		promptText := fmt.Sprintf(`Please summarize %s issues%s%s with the following structured approach:

## Overview
- Use issue_list to retrieve the issues matching the criteria
- Provide a high-level summary: total count, distribution by state/priority
- Identify key themes or patterns across the issues

## Categorization
Group issues into meaningful categories:
- **By Type**: bugs, features, enhancements, documentation, questions
- **By Priority**: critical, high, medium, low (based on labels or severity)
- **By Status**: in progress, blocked, needs review, ready to start
- **By Component**: if issues relate to different parts of the system

## Priority Issues
Highlight the most important or urgent issues:
- Critical bugs or security issues that need immediate attention
- High-priority features or enhancements
- Blockers preventing progress on other work
- Issues with approaching deadlines

## Recent Activity
- Summarize recently created issues (last week/month)
- Note issues with recent updates or comments
- Identify stale issues with no recent activity

## Trends & Insights
- Identify patterns: recurring bugs, frequently requested features
- Note areas of the codebase with many issues
- Highlight dependencies between issues
- Suggest issues that could be grouped or consolidated

## Recommendations
- Prioritize which issues to tackle first and why
- Suggest issues that could be quick wins (low effort, high impact)
- Identify issues that need more information or clarification
- Recommend issues that could be closed (duplicates, resolved, outdated)

Please use the issue_list tool to retrieve the issues, then provide a structured summary following these guidelines.`, state, repoInfo, filterInfo)

		description := fmt.Sprintf("Summary guidance for %s issues%s%s", state, repoInfo, filterInfo)

		return &mcp.GetPromptResult{
			Description: description,
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})
}

func registerDraftMRDescriptionPrompt(server *mcp.Server, f *cmdutil.Factory) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "draft_mr_description",
		Description: "Structured instructions for drafting a merge request description",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "repo",
				Description: "Repository in OWNER/REPO format (optional, uses current repo if not specified)",
				Required:    false,
			},
			{
				Name:        "source_branch",
				Description: "Source branch for the merge request (optional, uses current branch if not specified)",
				Required:    false,
			},
			{
				Name:        "target_branch",
				Description: "Target branch for the merge request (optional, uses default branch if not specified)",
				Required:    false,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		repo := req.Params.Arguments["repo"]
		repoInfo := ""
		if repo != "" {
			repoInfo = fmt.Sprintf(" in repository %s", repo)
		}

		sourceBranch := req.Params.Arguments["source_branch"]
		sourceBranchInfo := ""
		if sourceBranch != "" {
			sourceBranchInfo = fmt.Sprintf(" from branch %s", sourceBranch)
		}

		targetBranch := req.Params.Arguments["target_branch"]
		targetBranchInfo := ""
		if targetBranch != "" {
			targetBranchInfo = fmt.Sprintf(" to %s", targetBranch)
		}

		promptText := fmt.Sprintf(`Please draft a comprehensive merge request description%s%s%s with the following structured approach:

## Analyze Changes
- Use repo_compare to see the diff between source and target branches
- Review the commits included in this MR
- Understand the scope and purpose of the changes
- Identify the problem being solved or feature being added

## MR Description Structure

### Title
- Create a clear, concise title (under 70 characters)
- Start with a verb (Add, Fix, Update, Remove, Refactor, etc.)
- Accurately reflect the main change
- Example: "Add user authentication with JWT tokens"

### Summary
Write a brief overview (2-4 sentences) that answers:
- **What** changes are being made?
- **Why** are these changes needed?
- **How** do these changes solve the problem?

### Changes
List the key changes in bullet points:
- New features or functionality added
- Bugs fixed or issues resolved
- Code refactored or improved
- Dependencies added, updated, or removed
- Configuration changes
- Database schema changes (if applicable)

### Testing
Describe how the changes were tested:
- Unit tests added or updated
- Integration tests added or updated
- Manual testing performed
- Edge cases considered
- Test coverage impact

### Screenshots/Demo
If applicable, suggest including:
- Before/after screenshots for UI changes
- Example command outputs for CLI changes
- API request/response examples for API changes

### Breaking Changes
If there are breaking changes:
- Clearly mark them with ⚠️ or **BREAKING CHANGE**
- Explain what breaks and why
- Provide migration instructions

### Related Issues
- Link to related issues that this MR addresses
- Use "Closes #123" or "Fixes #456" for issues this MR resolves
- Reference related discussions or documentation

### Checklist
Add a checklist for reviewers:
- [ ] Code follows project style guidelines
- [ ] Tests added/updated and passing
- [ ] Documentation updated
- [ ] No breaking changes (or breaking changes documented)
- [ ] Ready for review

## Best Practices
- Write for your reviewers: help them understand the context
- Be specific and concrete, avoid vague language
- Include rationale for non-obvious decisions
- Link to relevant documentation, issues, or discussions
- Keep the tone professional and collaborative

Please use repo_compare and related tools to analyze the changes, then draft a well-structured merge request description following these guidelines.`, repoInfo, sourceBranchInfo, targetBranchInfo)

		description := fmt.Sprintf("Draft MR description guidance%s%s%s", repoInfo, sourceBranchInfo, targetBranchInfo)

		return &mcp.GetPromptResult{
			Description: description,
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})
}

func registerCreateReleaseNotesPrompt(server *mcp.Server, f *cmdutil.Factory) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "create_release_notes",
		Description: "Structured instructions for creating release notes",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "repo",
				Description: "Repository in OWNER/REPO format (optional, uses current repo if not specified)",
				Required:    false,
			},
			{
				Name:        "from_tag",
				Description: "Starting tag/version for the release notes (optional)",
				Required:    false,
			},
			{
				Name:        "to_tag",
				Description: "Ending tag/version for the release notes (optional, defaults to HEAD)",
				Required:    false,
			},
			{
				Name:        "milestone",
				Description: "Milestone to generate release notes for (alternative to from_tag/to_tag)",
				Required:    false,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		repo := req.Params.Arguments["repo"]
		repoInfo := ""
		if repo != "" {
			repoInfo = fmt.Sprintf(" for repository %s", repo)
		}

		fromTag := req.Params.Arguments["from_tag"]
		toTag := req.Params.Arguments["to_tag"]
		milestone := req.Params.Arguments["milestone"]

		var rangeInfo string
		if milestone != "" {
			rangeInfo = fmt.Sprintf(" for milestone %s", milestone)
		} else if fromTag != "" && toTag != "" {
			rangeInfo = fmt.Sprintf(" from %s to %s", fromTag, toTag)
		} else if fromTag != "" {
			rangeInfo = fmt.Sprintf(" since %s", fromTag)
		} else if toTag != "" {
			rangeInfo = fmt.Sprintf(" up to %s", toTag)
		}

		promptText := fmt.Sprintf(`Please create comprehensive release notes%s%s with the following structured approach:

## Gather Information
- Use repo_compare to see changes between versions (if using tags)
- Use mr_list to find merged MRs in the release period
- Use issue_list to find closed issues (if using milestone)
- Review commits included in this release
- Identify the version number and release date

## Release Notes Structure

### Version & Date
- Start with the version number (e.g., "v1.2.0" or "Release 1.2.0")
- Include the release date
- Add a one-line tagline describing the release theme

### Highlights
Start with 3-5 key highlights:
- Most important new features
- Major improvements or changes
- Critical bug fixes
- Performance enhancements
- Breaking changes (if any)

### New Features ✨
List new features added in this release:
- Use clear, user-focused descriptions
- Explain the benefit, not just the technical implementation
- Link to relevant documentation or issues
- Include examples or screenshots if helpful

### Improvements 🚀
List enhancements to existing features:
- Performance improvements
- UI/UX enhancements
- Better error messages
- Improved documentation
- Developer experience improvements

### Bug Fixes 🐛
List bugs fixed in this release:
- Describe the issue that was fixed
- Note the impact (who was affected, severity)
- Link to the issue tracker if applicable
- Group related fixes together

### Breaking Changes ⚠️
If there are breaking changes:
- Clearly mark this section
- Explain what breaks and why the change was necessary
- Provide migration instructions
- Include before/after examples
- Link to migration guides

### Deprecations
If any features are deprecated:
- List deprecated features or APIs
- Explain what to use instead
- Provide timeline for removal
- Include migration examples

### Security Updates 🔒
If there are security fixes:
- List security issues addressed (without exposing vulnerabilities)
- Note severity levels
- Recommend users to upgrade
- Credit security researchers if applicable

### Dependencies
Note significant dependency changes:
- Major version updates
- New dependencies added
- Dependencies removed
- Security updates in dependencies

### Contributors
Acknowledge contributors:
- Thank all contributors to this release
- Special mentions for significant contributions
- Link to contributor profiles

### Installation/Upgrade Instructions
Provide clear instructions:
- How to install this version
- How to upgrade from previous versions
- Any special steps required
- Link to full documentation

## Best Practices
- Write for end users, not developers (unless it's a developer tool)
- Use clear, jargon-free language
- Focus on user impact and benefits
- Keep it scannable with headers and bullet points
- Include links to issues, MRs, and documentation
- Use emojis sparingly for visual organization
- Follow semantic versioning principles

## Release Notes Tone
- Be enthusiastic about improvements
- Be transparent about breaking changes
- Be grateful to contributors
- Be helpful with migration guidance

Please use repo_compare, mr_list, issue_list, and related tools to gather information, then create comprehensive release notes following these guidelines.`, repoInfo, rangeInfo)

		description := fmt.Sprintf("Release notes guidance%s%s", repoInfo, rangeInfo)

		return &mcp.GetPromptResult{
			Description: description,
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})
}
