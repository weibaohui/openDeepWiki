package service

import (
	"context"
)

type docRepoService struct {
	parent *docService
}

func (s *docService) RepoService() *docRepoService {
	return &docRepoService{
		parent: s,
	}
}
func (s *docRepoService) Clone(ctx context.Context) error {

	return GitService().InitRepo(s.parent.repo)
}
func (s *docRepoService) GetRepoPath(ctx context.Context) (string, error) {

	return GitService().GetRepoPath(s.parent.repo)
}

func (s *docRepoService) GetRepoName(ctx context.Context) string {
	return s.parent.repo.Name
}
