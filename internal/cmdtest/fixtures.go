package cmdtest

// This file contains pre-built fixtures for common API responses.
// Tests can use these fixtures directly without having to construct test data.

// Merge Request Fixtures

// FixtureMROpen is a merge request in the opened state.
var FixtureMROpen = map[string]interface{}{
	"id":          100,
	"iid":         1,
	"title":       "Add new feature",
	"state":       "opened",
	"description": "This MR adds a new feature to improve user experience",
	"web_url":     "https://gitlab.com/test-owner/test-repo/-/merge_requests/1",
	"author": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
		"email":    "test-user@example.com",
	},
	"assignees": []interface{}{
		map[string]interface{}{
			"id":       2,
			"username": "reviewer",
			"name":     "Reviewer User",
		},
	},
	"source_branch":                 "feature/new-feature",
	"target_branch":                 "main",
	"merged_at":                     nil,
	"closed_at":                     nil,
	"created_at":                    "2024-01-01T10:00:00.000Z",
	"updated_at":                    "2024-01-02T14:30:00.000Z",
	"merge_status":                  "can_be_merged",
	"upvotes":                       3,
	"downvotes":                     0,
	"work_in_progress":              false,
	"draft":                         false,
	"has_conflicts":                 false,
	"blocking_discussions_resolved": true,
}

// FixtureMRMerged is a merge request that has been merged.
var FixtureMRMerged = map[string]interface{}{
	"id":          101,
	"iid":         2,
	"title":       "Fix critical bug",
	"state":       "merged",
	"description": "This MR fixes a critical bug in production",
	"web_url":     "https://gitlab.com/test-owner/test-repo/-/merge_requests/2",
	"author": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
		"email":    "test-user@example.com",
	},
	"assignees":     []interface{}{},
	"source_branch": "bugfix/critical-fix",
	"target_branch": "main",
	"merged_at":     "2024-01-03T16:45:00.000Z",
	"merged_by": map[string]interface{}{
		"id":       2,
		"username": "reviewer",
		"name":     "Reviewer User",
	},
	"closed_at":                     nil,
	"created_at":                    "2024-01-03T09:00:00.000Z",
	"updated_at":                    "2024-01-03T16:45:00.000Z",
	"merge_status":                  "merged",
	"upvotes":                       5,
	"downvotes":                     0,
	"work_in_progress":              false,
	"draft":                         false,
	"has_conflicts":                 false,
	"blocking_discussions_resolved": true,
}

// FixtureMRClosed is a merge request that has been closed without merging.
var FixtureMRClosed = map[string]interface{}{
	"id":          102,
	"iid":         3,
	"title":       "Experimental feature",
	"state":       "closed",
	"description": "This experimental feature was not accepted",
	"web_url":     "https://gitlab.com/test-owner/test-repo/-/merge_requests/3",
	"author": map[string]interface{}{
		"id":       3,
		"username": "contributor",
		"name":     "External Contributor",
		"email":    "contributor@example.com",
	},
	"assignees":     []interface{}{},
	"source_branch": "feature/experiment",
	"target_branch": "main",
	"merged_at":     nil,
	"closed_at":     "2024-01-04T12:00:00.000Z",
	"closed_by": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
	"created_at":                    "2024-01-02T08:00:00.000Z",
	"updated_at":                    "2024-01-04T12:00:00.000Z",
	"merge_status":                  "cannot_be_merged",
	"upvotes":                       0,
	"downvotes":                     2,
	"work_in_progress":              false,
	"draft":                         false,
	"has_conflicts":                 true,
	"blocking_discussions_resolved": false,
}

// Issue Fixtures

// FixtureIssueOpen is an issue in the opened state.
var FixtureIssueOpen = map[string]interface{}{
	"id":          200,
	"iid":         10,
	"title":       "Application crashes on startup",
	"state":       "opened",
	"description": "The application crashes when starting with certain configurations",
	"web_url":     "https://gitlab.com/test-owner/test-repo/-/issues/10",
	"author": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
		"email":    "test-user@example.com",
	},
	"assignees": []interface{}{
		map[string]interface{}{
			"id":       2,
			"username": "developer",
			"name":     "Developer User",
		},
	},
	"labels": []string{"bug", "priority::high"},
	"milestone": map[string]interface{}{
		"id":    5,
		"title": "v1.2.0",
	},
	"created_at":       "2024-01-05T09:00:00.000Z",
	"updated_at":       "2024-01-06T11:30:00.000Z",
	"closed_at":        nil,
	"due_date":         "2024-01-15",
	"upvotes":          2,
	"downvotes":        0,
	"user_notes_count": 3,
}

// FixtureIssueClosed is an issue that has been closed.
var FixtureIssueClosed = map[string]interface{}{
	"id":          201,
	"iid":         11,
	"title":       "Update documentation",
	"state":       "closed",
	"description": "Documentation needs to be updated with new API endpoints",
	"web_url":     "https://gitlab.com/test-owner/test-repo/-/issues/11",
	"author": map[string]interface{}{
		"id":       4,
		"username": "docs-writer",
		"name":     "Documentation Writer",
		"email":    "docs@example.com",
	},
	"assignees": []interface{}{
		map[string]interface{}{
			"id":       4,
			"username": "docs-writer",
			"name":     "Documentation Writer",
		},
	},
	"labels":     []string{"documentation"},
	"milestone":  nil,
	"created_at": "2024-01-03T14:00:00.000Z",
	"updated_at": "2024-01-05T16:00:00.000Z",
	"closed_at":  "2024-01-05T16:00:00.000Z",
	"closed_by": map[string]interface{}{
		"id":       4,
		"username": "docs-writer",
		"name":     "Documentation Writer",
	},
	"due_date":         nil,
	"upvotes":          1,
	"downvotes":        0,
	"user_notes_count": 1,
}

// Pipeline Fixtures

// FixturePipelineSuccess is a pipeline that completed successfully.
var FixturePipelineSuccess = map[string]interface{}{
	"id":      300,
	"iid":     50,
	"ref":     "main",
	"sha":     "abc123def456789012345678901234567890abcd",
	"status":  "success",
	"web_url": "https://gitlab.com/test-owner/test-repo/-/pipelines/300",
	"user": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
	"created_at":  "2024-01-06T10:00:00.000Z",
	"updated_at":  "2024-01-06T10:15:00.000Z",
	"started_at":  "2024-01-06T10:00:30.000Z",
	"finished_at": "2024-01-06T10:15:00.000Z",
	"duration":    870,
	"coverage":    "85.5",
}

// FixturePipelineFailed is a pipeline that failed.
var FixturePipelineFailed = map[string]interface{}{
	"id":      301,
	"iid":     51,
	"ref":     "feature/test-failure",
	"sha":     "def456789012345678901234567890abcdef1234",
	"status":  "failed",
	"web_url": "https://gitlab.com/test-owner/test-repo/-/pipelines/301",
	"user": map[string]interface{}{
		"id":       2,
		"username": "developer",
		"name":     "Developer User",
	},
	"created_at":  "2024-01-06T11:00:00.000Z",
	"updated_at":  "2024-01-06T11:08:00.000Z",
	"started_at":  "2024-01-06T11:00:30.000Z",
	"finished_at": "2024-01-06T11:08:00.000Z",
	"duration":    450,
	"coverage":    nil,
}

// FixturePipelineRunning is a pipeline currently running.
var FixturePipelineRunning = map[string]interface{}{
	"id":      302,
	"iid":     52,
	"ref":     "develop",
	"sha":     "1234567890abcdef1234567890abcdef12345678",
	"status":  "running",
	"web_url": "https://gitlab.com/test-owner/test-repo/-/pipelines/302",
	"user": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
	"created_at":  "2024-01-06T12:00:00.000Z",
	"updated_at":  "2024-01-06T12:05:00.000Z",
	"started_at":  "2024-01-06T12:00:30.000Z",
	"finished_at": nil,
	"duration":    nil,
	"coverage":    nil,
}

// FixturePipelinePending is a pipeline waiting to start.
var FixturePipelinePending = map[string]interface{}{
	"id":      303,
	"iid":     53,
	"ref":     "feature/new-pipeline",
	"sha":     "567890abcdef1234567890abcdef12345678901",
	"status":  "pending",
	"web_url": "https://gitlab.com/test-owner/test-repo/-/pipelines/303",
	"user": map[string]interface{}{
		"id":       3,
		"username": "contributor",
		"name":     "External Contributor",
	},
	"created_at":  "2024-01-06T13:00:00.000Z",
	"updated_at":  "2024-01-06T13:00:00.000Z",
	"started_at":  nil,
	"finished_at": nil,
	"duration":    nil,
	"coverage":    nil,
}

// Project Fixtures

// FixtureProject is a typical project.
var FixtureProject = map[string]interface{}{
	"id":                  400,
	"name":                "Test Repository",
	"path":                "test-repo",
	"description":         "A test repository for GitLab CLI development and testing",
	"web_url":             "https://gitlab.com/test-owner/test-repo",
	"ssh_url_to_repo":     "git@gitlab.com:test-owner/test-repo.git",
	"http_url_to_repo":    "https://gitlab.com/test-owner/test-repo.git",
	"path_with_namespace": "test-owner/test-repo",
	"namespace": map[string]interface{}{
		"id":        100,
		"name":      "test-owner",
		"path":      "test-owner",
		"kind":      "user",
		"full_path": "test-owner",
	},
	"visibility":             "public",
	"default_branch":         "main",
	"created_at":             "2023-06-01T00:00:00.000Z",
	"last_activity_at":       "2024-01-06T14:00:00.000Z",
	"star_count":             42,
	"forks_count":            7,
	"open_issues_count":      5,
	"archived":               false,
	"topics":                 []string{"cli", "gitlab", "go"},
	"readme_url":             "https://gitlab.com/test-owner/test-repo/-/blob/main/README.md",
	"avatar_url":             "",
	"ci_config_path":         ".gitlab-ci.yml",
	"shared_runners_enabled": true,
}

// FixtureProjectPrivate is a private project.
var FixtureProjectPrivate = map[string]interface{}{
	"id":                  401,
	"name":                "Private Project",
	"path":                "private-repo",
	"description":         "A private repository",
	"web_url":             "https://gitlab.com/test-owner/private-repo",
	"ssh_url_to_repo":     "git@gitlab.com:test-owner/private-repo.git",
	"http_url_to_repo":    "https://gitlab.com/test-owner/private-repo.git",
	"path_with_namespace": "test-owner/private-repo",
	"namespace": map[string]interface{}{
		"id":        100,
		"name":      "test-owner",
		"path":      "test-owner",
		"kind":      "user",
		"full_path": "test-owner",
	},
	"visibility":             "private",
	"default_branch":         "main",
	"created_at":             "2023-08-15T00:00:00.000Z",
	"last_activity_at":       "2024-01-05T10:00:00.000Z",
	"star_count":             0,
	"forks_count":            0,
	"open_issues_count":      2,
	"archived":               false,
	"topics":                 []string{},
	"readme_url":             "https://gitlab.com/test-owner/private-repo/-/blob/main/README.md",
	"avatar_url":             "",
	"ci_config_path":         ".gitlab-ci.yml",
	"shared_runners_enabled": true,
}

// User Fixtures

// FixtureUser is a typical user.
var FixtureUser = map[string]interface{}{
	"id":           1,
	"username":     "test-user",
	"name":         "Test User",
	"email":        "test-user@example.com",
	"state":        "active",
	"web_url":      "https://gitlab.com/test-user",
	"avatar_url":   "https://www.gravatar.com/avatar/test",
	"bio":          "Software developer and GitLab enthusiast",
	"location":     "San Francisco, CA",
	"created_at":   "2020-01-01T00:00:00.000Z",
	"is_admin":     false,
	"public_email": "test-user@example.com",
}

// FixtureUserAdmin is an admin user.
var FixtureUserAdmin = map[string]interface{}{
	"id":           10,
	"username":     "admin-user",
	"name":         "Admin User",
	"email":        "admin@example.com",
	"state":        "active",
	"web_url":      "https://gitlab.com/admin-user",
	"avatar_url":   "https://www.gravatar.com/avatar/admin",
	"bio":          "GitLab Administrator",
	"location":     "Remote",
	"created_at":   "2019-01-01T00:00:00.000Z",
	"is_admin":     true,
	"public_email": "admin@example.com",
}

// Release Fixtures

// FixtureRelease is a typical release.
var FixtureRelease = map[string]interface{}{
	"tag_name":    "v1.0.0",
	"name":        "Version 1.0.0",
	"description": "## What's Changed\n\n* Add new feature X\n* Fix bug Y\n* Improve performance Z\n\n**Full Changelog**: https://gitlab.com/test-owner/test-repo/-/compare/v0.9.0...v1.0.0",
	"created_at":  "2024-01-01T00:00:00.000Z",
	"released_at": "2024-01-01T12:00:00.000Z",
	"author": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
	"commit": map[string]interface{}{
		"id":         "abc123def456789012345678901234567890abcd",
		"short_id":   "abc123d",
		"title":      "Release v1.0.0",
		"message":    "Release v1.0.0\n\nFinal release for version 1.0.0",
		"created_at": "2024-01-01T00:00:00.000Z",
	},
	"assets": map[string]interface{}{
		"count": 2,
		"sources": []interface{}{
			map[string]string{
				"format": "zip",
				"url":    "https://gitlab.com/test-owner/test-repo/-/archive/v1.0.0/test-repo-v1.0.0.zip",
			},
			map[string]string{
				"format": "tar.gz",
				"url":    "https://gitlab.com/test-owner/test-repo/-/archive/v1.0.0/test-repo-v1.0.0.tar.gz",
			},
		},
		"links": []interface{}{},
	},
}

// Label Fixtures

// FixtureLabelBug is a bug label.
var FixtureLabelBug = map[string]interface{}{
	"id":                        500,
	"name":                      "bug",
	"color":                     "#d9534f",
	"description":               "Something isn't working correctly",
	"text_color":                "#FFFFFF",
	"open_issues_count":         3,
	"closed_issues_count":       15,
	"open_merge_requests_count": 1,
	"priority":                  nil,
}

// FixtureLabelFeature is a feature label.
var FixtureLabelFeature = map[string]interface{}{
	"id":                        501,
	"name":                      "feature",
	"color":                     "#5cb85c",
	"description":               "New feature or enhancement",
	"text_color":                "#FFFFFF",
	"open_issues_count":         7,
	"closed_issues_count":       25,
	"open_merge_requests_count": 3,
	"priority":                  nil,
}

// FixtureLabelPriority is a priority label.
var FixtureLabelPriority = map[string]interface{}{
	"id":                        502,
	"name":                      "priority::high",
	"color":                     "#f0ad4e",
	"description":               "High priority item",
	"text_color":                "#FFFFFF",
	"open_issues_count":         2,
	"closed_issues_count":       8,
	"open_merge_requests_count": 1,
	"priority":                  10,
}

// Snippet Fixtures

// FixtureSnippet is a typical code snippet.
var FixtureSnippet = map[string]interface{}{
	"id":          600,
	"title":       "Example Go Function",
	"file_name":   "example.go",
	"description": "A simple example function in Go",
	"visibility":  "public",
	"web_url":     "https://gitlab.com/-/snippets/600",
	"raw_url":     "https://gitlab.com/-/snippets/600/raw",
	"author": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
	"created_at": "2024-01-02T15:00:00.000Z",
	"updated_at": "2024-01-02T15:30:00.000Z",
	"files": []interface{}{
		map[string]interface{}{
			"path":    "example.go",
			"raw_url": "https://gitlab.com/-/snippets/600/raw/main/example.go",
		},
	},
}

// FixtureSnippetPrivate is a private snippet.
var FixtureSnippetPrivate = map[string]interface{}{
	"id":          601,
	"title":       "Private Configuration",
	"file_name":   "config.yml",
	"description": "Private configuration file",
	"visibility":  "private",
	"web_url":     "https://gitlab.com/-/snippets/601",
	"raw_url":     "https://gitlab.com/-/snippets/601/raw",
	"author": map[string]interface{}{
		"id":       1,
		"username": "test-user",
		"name":     "Test User",
	},
	"created_at": "2024-01-03T10:00:00.000Z",
	"updated_at": "2024-01-03T10:00:00.000Z",
	"files": []interface{}{
		map[string]interface{}{
			"path":    "config.yml",
			"raw_url": "https://gitlab.com/-/snippets/601/raw/main/config.yml",
		},
	},
}
