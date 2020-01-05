// Package fs implements an in-memory user store backed by a virtual filesystem.
package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"

	"github.com/shurcooL/users"
	"golang.org/x/net/webdav"
)

// NewStore creates an in-memory user store backed by
// a virtual filesystem root for storage.
func NewStore(root webdav.FileSystem) (*Store, error) {
	s := &Store{
		fs:    root,
		users: make(map[users.UserSpec]users.User),
	}
	err := s.load()
	if err != nil {
		return nil, err
	}
	return s, nil
}

type Store struct {
	mu    sync.Mutex
	fs    webdav.FileSystem
	users map[users.UserSpec]users.User
}

func (s *Store) load() error {
	f, err := s.fs.OpenFile(context.Background(), "users", os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	for {
		var u user
		err := dec.Decode(&u)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		user := u.User()
		s.users[user.UserSpec] = user
	}
	return nil
}

// Create creates the specified user.
// UserSpec must specify a valid (i.e., non-zero) user.
// It returns os.ErrExist if the user already exists.
func (s *Store) Create(ctx context.Context, user users.User) error {
	if user.UserSpec.ID == 0 || user.UserSpec.Domain == "" {
		return fmt.Errorf("Create: user ID 0 or empty domain are not valid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists.
	if _, ok := s.users[user.UserSpec]; ok {
		return os.ErrExist
	}

	// Commit to storage first, returning error on failure.
	f, err := s.fs.OpenFile(ctx, "users", os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(fromUser(user))
	if err != nil {
		return err
	}

	// Commit to memory second.
	s.users[user.UserSpec] = user

	return nil
}

// InsertByCanonicalMe inserts a user identified by the CanonicalMe
// field into the user store. If a user with the same CanonicalMe
// value doesn't exist yet, a new user is created. Otherwise,
// the existing user is updated. CanonicalMe must not be empty.
//
// The user ID must be 0 and domain must be non-empty.
// The returned user keeps the same domain and gets
// assigned a unique persistent non-zero ID.
func (s *Store) InsertByCanonicalMe(ctx context.Context, user users.User) (users.User, error) {
	if user.CanonicalMe == "" {
		return users.User{}, fmt.Errorf("InsertByCanonicalMe: user.CanonicalMe must not be empty")
	} else if user.UserSpec.ID != 0 || user.UserSpec.Domain == "" {
		return users.User{}, fmt.Errorf("InsertByCanonicalMe: user ID must be 0 and domain must be non-empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already exists,
	// otherwise determine user ID to use.
	user.UserSpec.ID = 1
	for _, u := range s.users {
		if u.CanonicalMe == user.CanonicalMe {
			user.UserSpec.ID = u.UserSpec.ID
			if reflect.DeepEqual(user, u) {
				// User already exists and doesn't need to be updated.
				return user, nil
			}
			// Need to update the user in store.
			break
		} else if u.UserSpec.Domain == user.UserSpec.Domain && u.UserSpec.ID >= user.UserSpec.ID {
			user.UserSpec.ID = u.UserSpec.ID + 1
		}
	}

	// Updating is done by appending to the end of users file,
	// since multiple entries with the same user spec are allowed,
	// and the latest entry takes precedence.
	//
	// TODO: Consider doing compaction/cleaning of multiple
	//       entries with the same user spec at some point.

	// Commit to storage first, returning error on failure.
	f, err := s.fs.OpenFile(ctx, "users", os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return users.User{}, err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(fromUser(user))
	if err != nil {
		return users.User{}, err
	}

	// Commit to memory second.
	s.users[user.UserSpec] = user

	return user, nil
}

// Get fetches the specified user.
func (s *Store) Get(_ context.Context, user users.UserSpec) (users.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[user]
	if !ok {
		return users.User{}, os.ErrNotExist
	}
	return u, nil
}
