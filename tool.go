package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mark3labs/mcp-go/mcp"
)

// GitCommit represents a single Git commit with its metadata.
type GitCommit struct {
	Hash    string `json:"hash"`    // The full SHA-1 hash of the commit
	Author  string `json:"author"`  // The author's name
	Date    string `json:"date"`    // The commit date in YYYY-MM-DD format
	Message string `json:"message"` // The commit message
}

// GitFileChange represents a change to a file in the Git repository.
type GitFileChange struct {
	Path       string `json:"path"`       // The path to the changed file
	ChangeType string `json:"changeType"` // Type of change: Added, Modified, or Deleted
}

// getRepository opens the git repository at the specified path
func getRepository() (*git.Repository, error) {
	dir := os.Getenv("CLIENT_WORKDIR")
	if dir == "" {
		dir = os.Getenv("WORKDIR")
	}
	return git.PlainOpen(dir)
}

// getCommitFromHead returns the commit N commits back from HEAD
func getCommitFromHead(r *git.Repository, commitsBack int) (*object.Commit, error) {
	ref, err := r.Head()
	if err != nil {
		return nil, fmt.Errorf("error getting HEAD: %v", err)
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		return nil, fmt.Errorf("error getting HEAD commit: %v", err)
	}

	// Walk back N commits
	for i := 0; i < commitsBack; i++ {
		if commit.NumParents() == 0 {
			return nil, fmt.Errorf("reached root commit before going back %d commits", commitsBack)
		}
		commit, err = commit.Parent(0)
		if err != nil {
			return nil, fmt.Errorf("error walking commit history: %v", err)
		}
	}

	return commit, nil
}

// formatDate formats a time.Time into YYYY-MM-DD format
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// registerTools registers all available Git tools
func (s *MCPServer) registerTools() {
	s.registerGitLogTool()
	s.registerGitChangedFilesTool()
	s.registerGitFileDiffTool()
	s.registerGitFileHistoryTool()
}

// registerGitLogTool creates and registers a tool for displaying Git commit history
func (s *MCPServer) registerGitLogTool() {
	tool := mcp.NewTool("git-log",
		mcp.WithDescription("Display git commit history with commit hash, author, date, and message."),
		mcp.WithNumber("limit",
			mcp.Description("Number of commits to display (default: 10)"),
		),
	)

	s.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := 10
		if val, ok := request.Params.Arguments["limit"].(float64); ok {
			limit = int(val)
		}

		r, err := getRepository()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error opening repository: %v", err)),
				},
				IsError: true,
			}, nil
		}

		ref, err := r.Head()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD reference: %v", err)),
				},
				IsError: true,
			}, nil
		}

		commits := []GitCommit{}
		commitIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting commit history: %v", err)),
				},
				IsError: true,
			}, nil
		}

		count := 0
		err = commitIter.ForEach(func(c *object.Commit) error {
			if count >= limit {
				return fmt.Errorf("reached limit")
			}

			commits = append(commits, GitCommit{
				Hash:    c.Hash.String(),
				Author:  c.Author.Name,
				Date:    formatDate(c.Author.When),
				Message: strings.Split(c.Message, "\n")[0],
			})
			count++
			return nil
		})

		if err != nil && err.Error() != "reached limit" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error iterating commits: %v", err)),
				},
				IsError: true,
			}, nil
		}

		var result strings.Builder
		result.WriteString("Git History:\n\n")
		for _, commit := range commits {
			result.WriteString(fmt.Sprintf("Commit: %s\n", commit.Hash))
			result.WriteString(fmt.Sprintf("Author: %s\n", commit.Author))
			result.WriteString(fmt.Sprintf("Date: %s\n", commit.Date))
			result.WriteString(fmt.Sprintf("Message: %s\n", commit.Message))
			result.WriteString("----------------------------------------\n")
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(result.String()),
			},
		}, nil
	})
}

// registerGitChangedFilesTool creates and registers a tool for listing changed files
func (s *MCPServer) registerGitChangedFilesTool() {
	tool := mcp.NewTool("git-changed-files",
		mcp.WithDescription("List files changed between HEAD and a specified number of commits back."),
		mcp.WithNumber("commits_back",
			mcp.Description("Number of commits to look back from HEAD"),
			mcp.Required(),
		),
	)

	s.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		commitsBack := getNumber(request.Params.Arguments, "commits_back")
		if commitsBack == 0 {
			commitsBack = 10
		}

		r, err := getRepository()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error opening repository: %v", err)),
				},
				IsError: true,
			}, nil
		}

		headRef, err := r.Head()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD reference: %v", err)),
				},
				IsError: true,
			}, nil
		}

		oldCommit, err := getCommitFromHead(r, commitsBack)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting old commit: %v", err)),
				},
				IsError: true,
			}, nil
		}

		headCommit, err := r.CommitObject(headRef.Hash())
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD commit: %v", err)),
				},
				IsError: true,
			}, nil
		}

		oldTree, err := oldCommit.Tree()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting old commit tree: %v", err)),
				},
				IsError: true,
			}, nil
		}

		headTree, err := headCommit.Tree()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD tree: %v", err)),
				},
				IsError: true,
			}, nil
		}

		changes := []GitFileChange{}
		patch, err := oldTree.Patch(headTree)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting changes: %v", err)),
				},
				IsError: true,
			}, nil
		}

		for _, filePatch := range patch.FilePatches() {
			from, to := filePatch.Files()
			if from == nil && to != nil {
				changes = append(changes, GitFileChange{
					Path:       to.Path(),
					ChangeType: "Added",
				})
			} else if from != nil && to == nil {
				changes = append(changes, GitFileChange{
					Path:       from.Path(),
					ChangeType: "Deleted",
				})
			} else if from != nil && to != nil {
				changes = append(changes, GitFileChange{
					Path:       to.Path(),
					ChangeType: "Modified",
				})
			}
		}

		var result strings.Builder
		result.WriteString(fmt.Sprintf("Files changed in the last %d commits:\n\n", commitsBack))
		for _, change := range changes {
			result.WriteString(fmt.Sprintf("[%s] %s\n", change.ChangeType, change.Path))
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(result.String()),
			},
		}, nil
	})
}

// registerGitFileDiffTool creates and registers a tool for showing file diffs
func (s *MCPServer) registerGitFileDiffTool() {
	tool := mcp.NewTool("git-file-diff",
		mcp.WithDescription("Show the diff of a specific file between HEAD and N commits back."),
		mcp.WithString("file_path",
			mcp.Description("Path to the file to show diff for"),
			mcp.Required(),
		),
		mcp.WithNumber("commits_back",
			mcp.Description("Number of commits to look back from HEAD"),
			mcp.Required(),
		),
	)

	s.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath := getString(request.Params.Arguments, "file_path")
		if filePath == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: file_path parameter is required"),
				},
				IsError: true,
			}, nil
		}

		commitsBack := getNumber(request.Params.Arguments, "commits_back")
		if commitsBack == 0 {
			commitsBack = 10
		}

		r, err := getRepository()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error opening repository: %v", err)),
				},
				IsError: true,
			}, nil
		}

		headRef, err := r.Head()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD reference: %v", err)),
				},
				IsError: true,
			}, nil
		}

		oldCommit, err := getCommitFromHead(r, commitsBack)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting old commit: %v", err)),
				},
				IsError: true,
			}, nil
		}

		headCommit, err := r.CommitObject(headRef.Hash())
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD commit: %v", err)),
				},
				IsError: true,
			}, nil
		}

		oldTree, err := oldCommit.Tree()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting old commit tree: %v", err)),
				},
				IsError: true,
			}, nil
		}

		headTree, err := headCommit.Tree()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD tree: %v", err)),
				},
				IsError: true,
			}, nil
		}

		patch, err := oldTree.Patch(headTree)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting changes: %v", err)),
				},
				IsError: true,
			}, nil
		}

		var fileDiff strings.Builder
		for _, filePatch := range patch.FilePatches() {
			from, to := filePatch.Files()
			if (from != nil && from.Path() == filePath) || (to != nil && to.Path() == filePath) {
				for _, chunk := range filePatch.Chunks() {
					switch chunk.Type() {
					case diff.Add:
						fileDiff.WriteString("+" + chunk.Content())
					case diff.Delete:
						fileDiff.WriteString("-" + chunk.Content())
					case diff.Equal:
						fileDiff.WriteString(" " + chunk.Content())
					}
				}
				break
			}
		}

		if fileDiff.Len() == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("No changes found for file: %s", filePath)),
				},
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fileDiff.String()),
			},
		}, nil
	})
}

// registerGitFileHistoryTool creates and registers a tool for showing file history
func (s *MCPServer) registerGitFileHistoryTool() {
	tool := mcp.NewTool("git-file-history",
		mcp.WithDescription("Show the complete history of a file with all commits and their diffs."),
		mcp.WithString("file_path",
			mcp.Description("Path to the file to show history for"),
			mcp.Required(),
		),
	)

	s.server.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath := getString(request.Params.Arguments, "file_path")
		if filePath == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: file_path parameter is required"),
				},
				IsError: true,
			}, nil
		}

		r, err := getRepository()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error opening repository: %v", err)),
				},
				IsError: true,
			}, nil
		}

		ref, err := r.Head()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting HEAD reference: %v", err)),
				},
				IsError: true,
			}, nil
		}

		var result strings.Builder
		var prevCommit *object.Commit

		commitIter, err := r.Log(&git.LogOptions{From: ref.Hash()})
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error getting commit history: %v", err)),
				},
				IsError: true,
			}, nil
		}

		err = commitIter.ForEach(func(c *object.Commit) error {
			result.WriteString(fmt.Sprintf("commit %s\n", c.Hash))
			result.WriteString(fmt.Sprintf("Author: %s <%s>\n", c.Author.Name, c.Author.Email))
			result.WriteString(fmt.Sprintf("Date:   %s\n\n", c.Author.When.Format("Mon Jan 2 15:04:05 2006 -0700")))
			result.WriteString(fmt.Sprintf("    %s\n\n", strings.ReplaceAll(c.Message, "\n", "\n    ")))

			if prevCommit != nil {
				currTree, err := c.Tree()
				if err != nil {
					return fmt.Errorf("error getting current tree: %v", err)
				}

				prevTree, err := prevCommit.Tree()
				if err != nil {
					return fmt.Errorf("error getting previous tree: %v", err)
				}

				patch, err := currTree.Patch(prevTree)
				if err != nil {
					return fmt.Errorf("error getting changes: %v", err)
				}

				for _, filePatch := range patch.FilePatches() {
					from, to := filePatch.Files()
					if (from != nil && from.Path() == filePath) || (to != nil && to.Path() == filePath) {
						if from == nil {
							result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
							result.WriteString("new file mode 100644\n")
						} else if to == nil {
							result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
							result.WriteString("deleted file mode 100644\n")
						} else {
							result.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
						}

						for _, chunk := range filePatch.Chunks() {
							switch chunk.Type() {
							case diff.Add:
								result.WriteString("+" + chunk.Content())
							case diff.Delete:
								result.WriteString("-" + chunk.Content())
							case diff.Equal:
								result.WriteString(" " + chunk.Content())
							}
						}
						break
					}
				}
			}

			prevCommit = c
			return nil
		})

		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("Error iterating commits: %v", err)),
				},
				IsError: true,
			}, nil
		}

		if result.Len() == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent(fmt.Sprintf("No history found for file: %s", filePath)),
				},
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(result.String()),
			},
		}, nil
	})
}

// getNumber gets a number from the arguments map
func getNumber(args map[string]interface{}, key string) int {
	if val, ok := args[key].(float64); ok {
		return int(val)
	}
	tmp, ok := args[key].(int)
	if ok {
		return tmp
	}
	return 0
}

// getString gets a string from the arguments map
func getString(args map[string]interface{}, key string) string {
	if val, ok := args[key].(string); ok {
		return val
	}
	return ""
}
