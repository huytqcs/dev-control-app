package runtime

import (
	"context"

	"devctl/internal/git"
)

// GitAdapter adapts internal/git.Service to the runtime package's GitProbe
// interface, translating git's own Status struct into runtime.GitState (the
// shape the rest of the runtime/API/WS layer already speaks).
type GitAdapter struct {
	svc *git.Service
}

func NewGitAdapter(svc *git.Service) *GitAdapter {
	return &GitAdapter{svc: svc}
}

func (a *GitAdapter) Status(ctx context.Context, repoPath string) (GitState, error) {
	s, err := a.svc.Status(ctx, repoPath)
	if err != nil {
		return GitState{}, err
	}
	return GitState{Branch: s.Branch, Dirty: s.Dirty, Ahead: s.Ahead, Behind: s.Behind}, nil
}
