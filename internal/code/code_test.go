package code_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/shurcooL/events/event"
	"github.com/shurcooL/home/internal/code"
	"github.com/shurcooL/notifications"
	"github.com/shurcooL/users"
)

func TestCode(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "code_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatal(err)
		}
	}()
	err = exec.Command("cp", "-R", filepath.Join("testdata", "repositories"), tempDir).Run()
	if err != nil {
		t.Fatal("cp -R failed:", err)
	}

	notifications := mockNotifications{}
	events := &mockEvents{}
	users := mockUsers{}
	service, err := code.NewService(filepath.Join(tempDir, "repositories"), notifications, events, users)
	if err != nil {
		t.Fatal("code.NewService:", err)
	}

	// Create a real HTTP server so we can git push to it.
	gitHandler, err := code.NewGitHandler(service, filepath.Join(tempDir, "repositories"), "", events, users, nil, func(req *http.Request) *http.Request { return req })
	if err != nil {
		t.Fatal("code.NewGitHandler:", err)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if ok := gitHandler.ServeGitMaybe(w, req); ok {
			return
		}
		t.Error("HTTP server got a non-git request")
		http.NotFound(w, req)
	}))
	defer httpServer.Close()

	// Test initial state.
	{
		want := []*code.Directory{
			{
				ImportPath:   "dmitri.shuralyov.com/emptyrepo",
				RepoRoot:     "dmitri.shuralyov.com/emptyrepo",
				RepoPackages: 0,
			},
			{
				ImportPath:   "dmitri.shuralyov.com/kebabcase",
				RepoRoot:     "dmitri.shuralyov.com/kebabcase",
				RepoPackages: 1,
				Package: &code.Package{
					Name:     "kebabcase",
					Synopsis: "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
					DocHTML: `<p>
Package kebabcase provides a parser for identifier names
using kebab-case naming convention.
</p>
<p>
Reference: <a href="https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch",
				Package: &code.Package{
					Name:     "scratch",
					Synopsis: "Package scratch is used for testing.",
					DocHTML: `<p>
Package scratch is used for testing.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/hello",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch",
				Package: &code.Package{
					Name: "main",
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image/jpeg",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
				Package: &code.Package{
					Name:     "jpeg",
					Synopsis: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
					DocHTML: `<p>
Package jpeg implements a tiny subset of a JPEG image decoder and encoder.
</p>
<p>
JPEG is defined in ITU-T T.81: <a href="http://www.w3.org/Graphics/JPEG/itu-t81.pdf">http://www.w3.org/Graphics/JPEG/itu-t81.pdf</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image/png",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
				Package: &code.Package{
					Name:     "png",
					Synopsis: "Package png implements a tiny subset of a PNG image decoder and encoder.",
					DocHTML: `<p>
Package png implements a tiny subset of a PNG image decoder and encoder.
</p>
<p>
The PNG specification is at <a href="http://www.w3.org/TR/PNG/">http://www.w3.org/TR/PNG/</a>.
</p>
`,
				},
			},
		}
		wantEvents := []event.Event(nil)

		got, err := service.ListDirectories(context.Background())
		if err != nil {
			t.Fatalf("service.ListDirectories: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Error("initial state: not equal")
		}
		if got, want := events.listAndReset(), wantEvents; !reflect.DeepEqual(got, want) {
			t.Errorf("initial state: events not equal:\n got: %+v\nwant: %+v", got, want)
		}
	}

	// Create a new empty repository.
	{
		err := service.CreateRepo(context.Background(), "dmitri.shuralyov.com/new/repo", "New repo is described here in some detail.")
		if err != nil {
			t.Fatal("service.CreateRepo:", err)
		}
	}

	// Test after empty repository created.
	{
		want := []*code.Directory{
			{
				ImportPath:   "dmitri.shuralyov.com/emptyrepo",
				RepoRoot:     "dmitri.shuralyov.com/emptyrepo",
				RepoPackages: 0,
			},
			{
				ImportPath:   "dmitri.shuralyov.com/kebabcase",
				RepoRoot:     "dmitri.shuralyov.com/kebabcase",
				RepoPackages: 1,
				Package: &code.Package{
					Name:     "kebabcase",
					Synopsis: "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
					DocHTML: `<p>
Package kebabcase provides a parser for identifier names
using kebab-case naming convention.
</p>
<p>
Reference: <a href="https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/new/repo",
				RepoRoot:     "dmitri.shuralyov.com/new/repo",
				RepoPackages: 0,
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch",
				Package: &code.Package{
					Name:     "scratch",
					Synopsis: "Package scratch is used for testing.",
					DocHTML: `<p>
Package scratch is used for testing.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/hello",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch",
				Package: &code.Package{
					Name: "main",
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image/jpeg",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
				Package: &code.Package{
					Name:     "jpeg",
					Synopsis: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
					DocHTML: `<p>
Package jpeg implements a tiny subset of a JPEG image decoder and encoder.
</p>
<p>
JPEG is defined in ITU-T T.81: <a href="http://www.w3.org/Graphics/JPEG/itu-t81.pdf">http://www.w3.org/Graphics/JPEG/itu-t81.pdf</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image/png",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
				Package: &code.Package{
					Name:     "png",
					Synopsis: "Package png implements a tiny subset of a PNG image decoder and encoder.",
					DocHTML: `<p>
Package png implements a tiny subset of a PNG image decoder and encoder.
</p>
<p>
The PNG specification is at <a href="http://www.w3.org/TR/PNG/">http://www.w3.org/TR/PNG/</a>.
</p>
`,
				},
			},
		}
		wantEvents := []event.Event{
			{
				Container: "dmitri.shuralyov.com/new/repo",
				Payload: event.Create{
					Type:        "repository",
					Description: "New repo is described here in some detail.",
				},
			},
		}

		got, err := service.ListDirectories(context.Background())
		if err != nil {
			t.Fatalf("service.ListDirectories: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Error("after empty repository created: not equal")
		}
		if got, want := events.listAndReset(), wantEvents; !reflect.DeepEqual(got, want) {
			t.Errorf("after empty repository created: events not equal:\n got: %+v\nwant: %+v", got, want)
		}
	}

	// Push a copy of scratch repository to the new repository.
	{
		cmd := exec.Command("git", "push", httpServer.URL+"/new/repo", "master:master")
		cmd.Dir = filepath.Join("testdata", "repositories", "dmitri.shuralyov.com", "scratch")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git push failed: %v\n%s", err, out)
		}
	}

	// Test after new repository pushed to.
	{
		want := []*code.Directory{
			{
				ImportPath:   "dmitri.shuralyov.com/emptyrepo",
				RepoRoot:     "dmitri.shuralyov.com/emptyrepo",
				RepoPackages: 0,
			},
			{
				ImportPath:   "dmitri.shuralyov.com/kebabcase",
				RepoRoot:     "dmitri.shuralyov.com/kebabcase",
				RepoPackages: 1,
				Package: &code.Package{
					Name:     "kebabcase",
					Synopsis: "Package kebabcase provides a parser for identifier names using kebab-case naming convention.",
					DocHTML: `<p>
Package kebabcase provides a parser for identifier names
using kebab-case naming convention.
</p>
<p>
Reference: <a href="https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers">https://en.wikipedia.org/wiki/Naming_convention_(programming)#Multiple-word_identifiers</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/new/repo",
				RepoRoot:     "dmitri.shuralyov.com/new/repo",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/new/repo",
				Package: &code.Package{
					Name:     "scratch",
					Synopsis: "Package scratch is used for testing.",
					DocHTML: `<p>
Package scratch is used for testing.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/new/repo/hello",
				RepoRoot:     "dmitri.shuralyov.com/new/repo",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/new/repo",
				Package: &code.Package{
					Name: "main",
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/new/repo/image",
				RepoRoot:     "dmitri.shuralyov.com/new/repo",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/new/repo/image",
			},
			{
				ImportPath:   "dmitri.shuralyov.com/new/repo/image/jpeg",
				RepoRoot:     "dmitri.shuralyov.com/new/repo",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/new/repo/image",
				Package: &code.Package{
					Name:     "jpeg",
					Synopsis: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
					DocHTML: `<p>
Package jpeg implements a tiny subset of a JPEG image decoder and encoder.
</p>
<p>
JPEG is defined in ITU-T T.81: <a href="http://www.w3.org/Graphics/JPEG/itu-t81.pdf">http://www.w3.org/Graphics/JPEG/itu-t81.pdf</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/new/repo/image/png",
				RepoRoot:     "dmitri.shuralyov.com/new/repo",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/new/repo/image",
				Package: &code.Package{
					Name:     "png",
					Synopsis: "Package png implements a tiny subset of a PNG image decoder and encoder.",
					DocHTML: `<p>
Package png implements a tiny subset of a PNG image decoder and encoder.
</p>
<p>
The PNG specification is at <a href="http://www.w3.org/TR/PNG/">http://www.w3.org/TR/PNG/</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch",
				Package: &code.Package{
					Name:     "scratch",
					Synopsis: "Package scratch is used for testing.",
					DocHTML: `<p>
Package scratch is used for testing.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/hello",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch",
				Package: &code.Package{
					Name: "main",
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image/jpeg",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
				Package: &code.Package{
					Name:     "jpeg",
					Synopsis: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
					DocHTML: `<p>
Package jpeg implements a tiny subset of a JPEG image decoder and encoder.
</p>
<p>
JPEG is defined in ITU-T T.81: <a href="http://www.w3.org/Graphics/JPEG/itu-t81.pdf">http://www.w3.org/Graphics/JPEG/itu-t81.pdf</a>.
</p>
`,
				},
			},
			{
				ImportPath:   "dmitri.shuralyov.com/scratch/image/png",
				RepoRoot:     "dmitri.shuralyov.com/scratch",
				RepoPackages: 4,
				LicenseRoot:  "dmitri.shuralyov.com/scratch/image",
				Package: &code.Package{
					Name:     "png",
					Synopsis: "Package png implements a tiny subset of a PNG image decoder and encoder.",
					DocHTML: `<p>
Package png implements a tiny subset of a PNG image decoder and encoder.
</p>
<p>
The PNG specification is at <a href="http://www.w3.org/TR/PNG/">http://www.w3.org/TR/PNG/</a>.
</p>
`,
				},
			},
		}
		wantEvents := []event.Event{
			{
				Container: "dmitri.shuralyov.com/new/repo",
				Payload: event.Create{
					Type: "branch",
					Name: "master",
				},
			},
			{
				Container: "dmitri.shuralyov.com/new/repo",
				Payload: event.Create{
					Type:        "package",
					Description: "Package scratch is used for testing.",
				},
			},
			{
				Container: "dmitri.shuralyov.com/new/repo/hello",
				Payload: event.Create{
					Type: "package",
				},
			},
			{
				Container: "dmitri.shuralyov.com/new/repo/image/jpeg",
				Payload: event.Create{
					Type:        "package",
					Description: "Package jpeg implements a tiny subset of a JPEG image decoder and encoder.",
				},
			},
			{
				Container: "dmitri.shuralyov.com/new/repo/image/png",
				Payload: event.Create{
					Type:        "package",
					Description: "Package png implements a tiny subset of a PNG image decoder and encoder.",
				},
			},
		}

		got, err := service.ListDirectories(context.Background())
		if err != nil {
			t.Fatalf("service.ListDirectories: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Error("after new repository pushed to: not equal")
		}
		if got, want := events.listAndReset(), wantEvents; !reflect.DeepEqual(got, want) {
			t.Errorf("after new repository pushed to: events not equal:\n got: %+v\nwant: %+v", got, want)
		}
	}
}

type mockNotifications struct{ notifications.Service }

func (mockNotifications) Subscribe(context.Context, notifications.RepoSpec, string, uint64, []users.UserSpec) error {
	return nil
}

type mockEvents struct {
	mu     sync.Mutex
	events []event.Event
}

func (m *mockEvents) Log(ctx context.Context, event event.Event) error {
	m.mu.Lock()
	m.events = append(m.events, event)
	m.mu.Unlock()
	return nil
}

// listAndReset returns all events with Container and Payload fields populated,
// and resets the service to be empty. It's meant for testing purposes.
func (m *mockEvents) listAndReset() []event.Event {
	var events []event.Event
	m.mu.Lock()
	for _, e := range m.events {
		events = append(events, event.Event{
			Container: e.Container,
			Payload:   e.Payload,
		})
	}
	m.events = nil
	m.mu.Unlock()
	return events
}

type mockUsers struct{ users.Service }

func (mockUsers) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	if user.ID == 0 {
		return users.User{}, fmt.Errorf("user %v not found", user)
	}
	return users.User{
		UserSpec:  user,
		Login:     fmt.Sprintf("%d@%s", user.ID, user.Domain),
		AvatarURL: "https://secure.gravatar.com/avatar?d=mm&f=y&s=96",
		HTMLURL:   "",
		SiteAdmin: user == users.UserSpec{ID: 1, Domain: "example.org"}, // For CreateRepo.
	}, nil
}

func (mockUsers) GetAuthenticatedSpec(_ context.Context) (users.UserSpec, error) {
	return users.UserSpec{ID: 1, Domain: "example.org"}, nil
}

func (m mockUsers) GetAuthenticated(ctx context.Context) (users.User, error) {
	userSpec, err := m.GetAuthenticatedSpec(ctx)
	if err != nil {
		return users.User{}, err
	}
	if userSpec.ID == 0 {
		return users.User{}, nil
	}
	return m.Get(ctx, userSpec)
}
