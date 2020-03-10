// Package gerritapi implements a read-only change.Service using Gerrit API client.
package gerritapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"dmitri.shuralyov.com/state"
	"github.com/andygrunwald/go-gerrit"
	"github.com/shurcooL/home/internal/exp/service/change"
	"github.com/shurcooL/issues"
	"github.com/shurcooL/users"
)

// NewService creates a Gerrit-backed issues.Service using given Gerrit client.
// client must be non-nil.
func NewService(client *gerrit.Client) change.Service {
	return service{
		cl:     client,
		domain: client.BaseURL().Host,
	}
}

type service struct {
	cl     *gerrit.Client
	domain string
}

func (s service) List(ctx context.Context, repo string, opt change.ListOptions) ([]change.Change, error) {
	project := project(repo)
	var query string
	switch opt.Filter {
	case change.FilterOpen:
		query = fmt.Sprintf("project:%s status:open", project)
	case change.FilterClosedMerged:
		// "status:closed" is equivalent to "(status:abandoned OR status:merged)".
		query = fmt.Sprintf("project:%s status:closed", project)
	case change.FilterAll:
		query = fmt.Sprintf("project:%s", project)
	}
	cs, resp, err := s.cl.Changes.QueryChanges(&gerrit.QueryChangeOptions{
		QueryOptions: gerrit.QueryOptions{
			Query: []string{query},
			Limit: 25,
		},
		ChangeOptions: gerrit.ChangeOptions{
			AdditionalFields: []string{"DETAILED_ACCOUNTS", "MESSAGES"},
		},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	var is []change.Change
	for _, chg := range *cs {
		if chg.Status == "DRAFT" {
			continue
		}
		var labels []issues.Label
		for _, hashtag := range chg.Hashtags {
			labels = append(labels, issues.Label{
				Name:  hashtag,
				Color: issues.RGB{R: 0xed, G: 0xed, B: 0xed}, // A default light gray.
			})
		}
		is = append(is, change.Change{
			ID:        uint64(chg.Number),
			State:     changeState(chg.Status),
			Title:     chg.Subject,
			Labels:    labels,
			Author:    s.gerritUser(chg.Owner),
			CreatedAt: chg.Created.Time,
			Replies:   len(chg.Messages),
		})
	}
	//sort.Sort(sort.Reverse(byID(is))) // For some reason, IDs don't completely line up with created times.
	sort.Slice(is, func(i, j int) bool {
		return is[i].CreatedAt.After(is[j].CreatedAt)
	})
	return is, nil
}

func (s service) Count(_ context.Context, repo string, opt change.ListOptions) (uint64, error) {
	// TODO.
	return 0, nil
}

func (s service) Get(ctx context.Context, repo string, id uint64) (change.Change, error) {
	project := project(repo)
	chg, resp, err := s.cl.Changes.GetChange(fmt.Sprintf("%s~%d", project, id), &gerrit.ChangeOptions{
		AdditionalFields: []string{"DETAILED_ACCOUNTS", "MESSAGES", "ALL_REVISIONS"},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return change.Change{}, os.ErrNotExist
		}
		return change.Change{}, err
	}
	if chg.Status == "DRAFT" {
		return change.Change{}, os.ErrNotExist
	}
	var labels []issues.Label
	for _, hashtag := range chg.Hashtags {
		labels = append(labels, issues.Label{
			Name:  hashtag,
			Color: issues.RGB{R: 0xed, G: 0xed, B: 0xed}, // A default light gray.
		})
	}
	return change.Change{
		ID:           id,
		State:        changeState(chg.Status),
		Title:        chg.Subject,
		Labels:       labels,
		Author:       s.gerritUser(chg.Owner),
		CreatedAt:    chg.Created.Time,
		Replies:      len(chg.Messages),
		Commits:      len(chg.Revisions),
		ChangedFiles: 0, // TODO.
	}, nil
}

func changeState(status string) state.Change {
	switch status {
	case "NEW":
		return state.ChangeOpen
	case "ABANDONED":
		return state.ChangeClosed
	case "MERGED":
		return state.ChangeMerged
	case "DRAFT":
		panic("not sure how to deal with DRAFT status")
	default:
		panic("unreachable")
	}
}

func (s service) ListCommits(ctx context.Context, repo string, id uint64) ([]change.Commit, error) {
	project := project(repo)
	chg, resp, err := s.cl.Changes.GetChange(fmt.Sprintf("%s~%d", project, id), &gerrit.ChangeOptions{
		AdditionalFields: []string{"DETAILED_ACCOUNTS", "ALL_REVISIONS"},
		//AdditionalFields: []string{"ALL_REVISIONS", "ALL_COMMITS"}, // TODO: Consider using git committer/author instead...
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	if chg.Status == "DRAFT" {
		return nil, os.ErrNotExist
	}
	commits := make([]change.Commit, len(chg.Revisions))
	for sha, r := range chg.Revisions {
		commits[r.Number-1] = change.Commit{
			SHA:     sha,
			Message: fmt.Sprintf("Patch Set %d", r.Number),
			// TODO: r.Uploader and r.Created describe the committer, not author.
			Author:     s.gerritUser(r.Uploader),
			AuthorTime: r.Created.Time,
		}
	}
	return commits, nil
}

func (s service) GetDiff(ctx context.Context, repo string, id uint64, opt *change.GetDiffOptions) ([]byte, error) {
	project := project(repo)
	switch opt {
	case nil:
		diff, resp, err := s.cl.Changes.GetPatch(fmt.Sprintf("%s~%d", project, id), "current", nil)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, os.ErrNotExist
			}
			return nil, err
		}
		return []byte(*diff), nil
	default:
		chg, resp, err := s.cl.Changes.GetChange(fmt.Sprintf("%s~%d", project, id), &gerrit.ChangeOptions{
			AdditionalFields: []string{"ALL_REVISIONS"},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, os.ErrNotExist
			}
			return nil, err
		}
		if chg.Status == "DRAFT" {
			return nil, os.ErrNotExist
		}
		r, ok := chg.Revisions[opt.Commit]
		if !ok {
			return nil, os.ErrNotExist
		}
		var base string
		switch r.Number {
		case 1:
			base = ""
		default:
			base = fmt.Sprint(r.Number - 1)
		}
		files, _, err := s.cl.Changes.ListFiles(fmt.Sprintf("%s~%d", project, id), opt.Commit, &gerrit.FilesOptions{
			Base: base,
		})
		if err != nil {
			return nil, err
		}
		var sortedFiles []string
		for file := range files {
			sortedFiles = append(sortedFiles, file)
		}
		sort.Strings(sortedFiles)
		var diff string
		for _, file := range sortedFiles {
			diffInfo, _, err := s.cl.Changes.GetDiff(fmt.Sprintf("%s~%d", project, id), opt.Commit, file, &gerrit.DiffOptions{
				Base:    base,
				Context: "5",
			})
			if err != nil {
				return nil, err
			}
			diff += strings.Join(diffInfo.DiffHeader, "\n") + "\n"
			for i, c := range diffInfo.Content {
				if i == 0 {
					diff += "@@ -154,6 +154,7 @@\n" // TODO.
				}
				switch {
				case len(c.AB) > 0:
					if len(c.AB) <= 10 {
						for _, line := range c.AB {
							diff += " " + line + "\n"
						}
					} else {
						switch i {
						case 0:
							for _, line := range c.AB[len(c.AB)-5:] {
								diff += " " + line + "\n"
							}
						default:
							for _, line := range c.AB[:5] {
								diff += " " + line + "\n"
							}
							diff += "@@ -154,6 +154,7 @@\n" // TODO.
							for _, line := range c.AB[len(c.AB)-5:] {
								diff += " " + line + "\n"
							}
						case len(diffInfo.Content) - 1:
							for _, line := range c.AB[:5] {
								diff += " " + line + "\n"
							}
						}
					}
				case len(c.A) > 0 || len(c.B) > 0:
					for _, line := range c.A {
						diff += "-" + line + "\n"
					}
					for _, line := range c.B {
						diff += "+" + line + "\n"
					}
				}
			}
		}
		return []byte(diff), nil
	}
}

func (s service) ListTimeline(ctx context.Context, repo string, id uint64, opt *change.ListTimelineOptions) ([]interface{}, error) {
	// TODO: Pagination. Respect opt.Start and opt.Length, if given.

	project := project(repo)
	chg, resp, err := s.cl.Changes.GetChangeDetail(fmt.Sprintf("%s~%d", project, id), &gerrit.ChangeOptions{
		AdditionalFields: []string{"ALL_REVISIONS"},
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	commit, _, err := s.cl.Changes.GetCommit(fmt.Sprintf("%s~%d", project, id), "current", nil)
	if err != nil {
		return nil, err
	}
	comments, _, err := s.cl.Changes.ListChangeComments(fmt.Sprintf("%s~%d", project, id))
	if err != nil {
		return nil, err
	}
	var timeline []interface{}
	timeline = append(timeline, change.Comment{ // CL description.
		ID:        "0",
		User:      s.gerritUser(chg.Owner),
		CreatedAt: chg.Created.Time,
		Body:      commitMessageBody(commit.Message),
		Editable:  false,
	})
	var mergedRevisionSHA string               // Set to merged revision SHA when a change.MergedEvent event is encountered.
	var labelChanges = make(map[time.Time]int) // Map of times when a label was changed by someone. Time -> AccountID.
	for idx, message := range chg.Messages {
		if strings.HasPrefix(message.Tag, "autogenerated:") {
		Outer:
			switch message.Tag[len("autogenerated:"):] {
			case "gerrit:merged":
				timeline = append(timeline, change.TimelineItem{
					Actor:     s.gerritUser(message.Author),
					CreatedAt: message.Date.Time,
					Payload: change.MergedEvent{
						CommitID: chg.CurrentRevision,
						RefName:  chg.Branch,
					},
				})
				mergedRevisionSHA = chg.CurrentRevision
			case "gerrit:newPatchSet":
				// Parse a new patchset message, check if it has comments.
				body, err := parsePSMessage(message.Message, message.RevisionNumber)
				if err != nil {
					return nil, err
				}
				if body == "" {
					// No body means no comments.
					break
				}
				var cs []change.InlineComment
				for file, comments := range *comments {
					for _, c := range comments {
						if c.Updated.Equal(message.Date.Time) {
							cs = append(cs, change.InlineComment{
								File: file,
								Line: c.Line,
								Body: c.Message,
							})
						}
					}
				}
				sort.Slice(cs, func(i, j int) bool {
					if cs[i].File == cs[j].File {
						return cs[i].Line < cs[j].Line
					}
					return cs[i].File < cs[j].File
				})
				timeline = append(timeline, change.Review{
					ID:        fmt.Sprint(idx), // TODO: message.ID is not uint64; e.g., "bfba753d015916303152305cee7152ea7a112fe0".
					User:      s.gerritUser(message.Author),
					CreatedAt: message.Date.Time,
					Body:      body,
					Editable:  false,
					Comments:  cs,
				})
			case "gerrit:abandon":
				timeline = append(timeline, change.TimelineItem{
					Actor:     s.gerritUser(message.Author),
					CreatedAt: message.Date.Time,
					Payload:   change.ClosedEvent{},
				})
				if message.Message == "Abandoned" {
					// An abandon reason wasn't provided.
					break
				}
				timeline = append(timeline, change.Comment{
					ID:        fmt.Sprint(idx), // TODO: message.ID is not uint64; e.g., "bfba753d015916303152305cee7152ea7a112fe0".
					User:      s.gerritUser(message.Author),
					CreatedAt: message.Date.Time,
					Body:      strings.TrimPrefix(message.Message, "Abandoned\n\n"),
					Editable:  false,
				})
			case "gerrit:restore":
				timeline = append(timeline, change.TimelineItem{
					Actor:     s.gerritUser(message.Author),
					CreatedAt: message.Date.Time,
					Payload:   change.ReopenedEvent{},
				})
			case "gerrit:setHashtag":
				var payload interface{} // One of change.LabeledEvent or change.UnlabeledEvent.
				switch {
				case strings.HasPrefix(message.Message, "Hashtag added: "):
					hashtag := message.Message[len("Hashtag added: "):]
					payload = change.LabeledEvent{
						Label: issues.Label{
							Name:  hashtag,
							Color: issues.RGB{R: 0xed, G: 0xed, B: 0xed}, // A default light gray.
						},
					}
				case strings.HasPrefix(message.Message, "Hashtag removed: "):
					hashtag := message.Message[len("Hashtag removed: "):]
					payload = change.UnlabeledEvent{
						Label: issues.Label{
							Name:  hashtag,
							Color: issues.RGB{R: 0xed, G: 0xed, B: 0xed}, // A default light gray.
						},
					}
				default:
					log.Printf("unknown setHashtag message: %q", message.Message)
					break Outer
				}
				timeline = append(timeline, change.TimelineItem{
					Actor:     s.gerritUser(message.Author),
					CreatedAt: message.Date.Time,
					Payload:   payload,
				})
			}
			continue
		}
		labels, body, ok := parseMessage(message.Message)
		if !ok {
			continue
		}
		if labels != "" {
			labelChanges[message.Date.Time] = message.Author.AccountID
		}
		var cs []change.InlineComment
		for file, comments := range *comments {
			for _, c := range comments {
				if c.Updated.Equal(message.Date.Time) {
					cs = append(cs, change.InlineComment{
						File: file,
						Line: c.Line,
						Body: c.Message,
					})
				}
			}
		}
		sort.Slice(cs, func(i, j int) bool {
			if cs[i].File == cs[j].File {
				return cs[i].Line < cs[j].Line
			}
			return cs[i].File < cs[j].File
		})
		reviewState := reviewState(labels)
		if body == "" && len(cs) == 0 && reviewState == state.ReviewNoScore && labels != "" {
			// Skip an empty comment that, e.g., just sets a Run-TryBot+1 label.
			continue
		}
		timeline = append(timeline, change.Review{
			ID:        fmt.Sprint(idx), // TODO: message.ID is not uint64; e.g., "bfba753d015916303152305cee7152ea7a112fe0".
			User:      s.gerritUser(message.Author),
			CreatedAt: message.Date.Time,
			State:     reviewState,
			Body:      body,
			Editable:  false,
			Comments:  cs,
		})
	}
	for sha, r := range chg.Revisions {
		if r.Number == 1 || sha == mergedRevisionSHA {
			// Skip first revision because it's equal to the change itself, and
			// skip merged revision because it's equal to the change.MergedEvent.
			continue
		}
		timeline = append(timeline, change.TimelineItem{
			Actor:     s.gerritUser(r.Uploader),
			CreatedAt: r.Created.Time,
			Payload: change.CommitEvent{
				SHA:     sha,
				Subject: fmt.Sprintf("Patch Set %d", r.Number),
			},
		})
	}
	var reviewers = make(map[int]struct{}) // Set of reviewers during ReviewerUpdates iteration. Key is AccountID.
	for _, ru := range chg.ReviewerUpdates {
		switch ru.State {
		case "REVIEWER":
			reviewers[ru.Reviewer.AccountID] = struct{}{}
			if ru.UpdatedBy.AccountID == ru.Reviewer.AccountID &&
				labelChanges[ru.Updated.Time] == ru.UpdatedBy.AccountID {
				// Skip because it was an implicit add-self-reviewer due to label change.
				continue
			}
			timeline = append(timeline, change.TimelineItem{
				Actor:     s.gerritUser(ru.UpdatedBy),
				CreatedAt: ru.Updated.Time,
				Payload: change.ReviewRequestedEvent{
					RequestedReviewer: s.gerritUser(ru.Reviewer),
				},
			})
		case "CC", "REMOVED":
			if _, ok := reviewers[ru.Reviewer.AccountID]; !ok {
				// Skip because they weren't a reviewer.
				continue
			}
			delete(reviewers, ru.Reviewer.AccountID)
			timeline = append(timeline, change.TimelineItem{
				Actor:     s.gerritUser(ru.UpdatedBy),
				CreatedAt: ru.Updated.Time,
				Payload: change.ReviewRequestRemovedEvent{
					RequestedReviewer: s.gerritUser(ru.Reviewer),
				},
			})
		}
	}
	return timeline, nil
}

func parseMessage(m string) (labels string, body string, ok bool) {
	// "Patch Set ".
	if !strings.HasPrefix(m, "Patch Set ") {
		return "", "", false
	}
	m = m[len("Patch Set "):]

	// "123".
	i := strings.IndexFunc(m, func(c rune) bool { return !unicode.IsNumber(c) })
	if i == -1 {
		return "", "", false
	}
	m = m[i:]

	// ":".
	if len(m) < 1 || m[0] != ':' {
		return "", "", false
	}
	m = m[1:]

	switch i = strings.IndexByte(m, '\n'); i {
	case -1:
		labels = m
	default:
		labels = m[:i]
		body = m[i+1:]
	}

	if labels != "" {
		// " ".
		if len(labels) < 1 || labels[0] != ' ' {
			return "", "", false
		}
		labels = labels[1:]
	}

	if body != "" {
		// "\n".
		if len(body) < 1 || body[0] != '\n' {
			return "", "", false
		}
		body = body[1:]
	}

	return labels, body, true
}

// parsePSMessage parses an autogenerated:gerrit:newPatchSet
// message and returns its body, if any.
func parsePSMessage(m string, revisionNumber int) (body string, _ error) {
	// "Uploaded patch set ".
	if !strings.HasPrefix(m, "Uploaded patch set ") {
		if strings.HasPrefix(m, "Patch Set ") {
			// No body. Maybe just the commit message changed.
			return "", nil
		}
		return "", fmt.Errorf("unexpected format")
	}
	m = m[len("Uploaded patch set "):]

	// Revision number, e.g., "123".
	i := matchNumber(m, revisionNumber)
	if i == -1 {
		return "", fmt.Errorf("unexpected format")
	}
	m = m[i:]

	switch {
	// ".".
	case len(m) >= 1 && m[0] == '.':
		m = m[1:]
	// ":".
	case len(m) >= 1 && m[0] == ':':
		m = m[1:]

		// " Run-TryBot+1."
		i := strings.IndexByte(m, '.')
		if i == -1 {
			return "", fmt.Errorf("unexpected format")
		}
		m = m[i+1:]
	default:
		return "", fmt.Errorf("unexpected format")
	}

	if m == "" {
		// No body.
		return "", nil
	}

	switch {
	// "\n\n".
	case strings.HasPrefix(m, "\n\n"):
		m = m[len("\n\n"):]
	// "\n".
	case strings.HasPrefix(m, "\n"):
		m = m[len("\n"):]
	default:
		return "", fmt.Errorf("unexpected format")
	}

	// The remainer is the body.
	return m, nil
}

// matchNumber returns the index after number in s,
// or -1 if number is not immediately present in s.
func matchNumber(s string, number int) int {
	a := strconv.Itoa(number)
	if !strings.HasPrefix(s, a) {
		return -1
	}
	return len(a)
}

func reviewState(labels string) state.Review {
	for _, label := range strings.Split(labels, " ") {
		switch label {
		case "Code-Review+2":
			return state.ReviewPlus2
		case "Code-Review+1":
			return state.ReviewPlus1
		case "Code-Review-1":
			return state.ReviewMinus1
		case "Code-Review-2":
			return state.ReviewMinus2
		}
	}
	return state.ReviewNoScore
}

func (service) EditComment(_ context.Context, repo string, id uint64, cr change.CommentRequest) (change.Comment, error) {
	return change.Comment{}, fmt.Errorf("EditComment: not implemented")
}

func (s service) gerritUser(user gerrit.AccountInfo) users.User {
	var avatarURL string
	for _, avatar := range user.Avatars {
		if avatar.Height == 100 {
			avatarURL = avatar.URL
		}
	}
	return users.User{
		UserSpec: users.UserSpec{
			ID:     uint64(user.AccountID),
			Domain: s.domain,
		},
		Login: user.Name, //user.Username, // TODO.
		Name:  user.Name,
		//Email:     user.Email,
		AvatarURL: avatarURL,
	}
}

// commitMessageBody returns the body from a commit message.
// It trims off the subject from start, and headers from end.
func commitMessageBody(s string) string {
	i := strings.Index(s, "\n\n")
	if i == -1 {
		return ""
	}
	i += len("\n\n")
	j := strings.LastIndex(s, "\n\n")
	if i > j {
		// Only a subject and headers, no body.
		return ""
	}
	return s[i:j]
}

func project(repo string) string {
	i := strings.IndexByte(repo, '/')
	if i == -1 {
		return ""
	}
	return repo[i+1:]
}

// gerritChangeThreadType is the notification thread type for Gerrit changes.
const gerritChangeThreadType = "Change"

// ThreadType returns the notification thread type for this service.
func (service) ThreadType(repo string) string { return gerritChangeThreadType }
