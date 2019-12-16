package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"sourcegraph.com/sourcegraph/go-vcs/vcs"
	"sourcegraph.com/sourcegraph/go-vcs/vcs/gitcmd"
)

// foreachRef walks over lines sent to pre-receive git hook,
// and calls f on each.
// It returns early if there's an error or f returns a non-nil error.
func foreachRef(stdin io.Reader, f func(shaOld, shaNew, refName string) error) error {
	s := bufio.NewScanner(stdin)
	for s.Scan() {
		p := strings.Split(s.Text(), " ")
		if len(p) != 3 {
			return fmt.Errorf("line %q isn't like 'sha1-old SP sha1-new SP refname': got %d parts, want 3", s.Text(), len(p))
		}
		shaOld, shaNew, refName := p[0], p[1], p[2]
		err := f(shaOld, shaNew, refName)
		if err != nil {
			return err
		}
	}
	return s.Err()
}

// foreachCommit walks commits in git repo in current directory
// from shaOld to shaNew (in that order), and calls f on each one.
// It returns early if there's an error or f returns a non-nil error.
func foreachCommit(shaOld, shaNew string, f func(r vcs.Repository, c *vcs.Commit) error) error {
	r, err := gitcmd.Open(".")
	if err != nil {
		return err
	}
	defer r.Close()
	commits, _, err := r.Commits(vcs.CommitsOptions{
		Head:    vcs.CommitID(shaNew),
		Base:    vcs.CommitID(shaOld),
		NoTotal: true,
	})
	if err != nil {
		return err
	}
	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]
		if c.Committer == nil {
			return fmt.Errorf("commit %q has no committer", c.ID)
		}
		err := f(r, c)
		if err != nil {
			return err
		}
	}
	return nil
}

// getCommit returns a commit with id from r
// and checks it has a non-nil Committer field.
func getCommit(r vcs.Repository, id vcs.CommitID) (*vcs.Commit, error) {
	c, err := r.GetCommit(id)
	if err != nil {
		return nil, err
	} else if c.Committer == nil {
		return nil, fmt.Errorf("commit %q has no committer", c.ID)
	}
	return c, nil
}
