package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

// PlatformAccountLink is a learner-confirmed record linking a platform user to
// the workspace learner (spec #66 Q36). Platform users are identified only by
// {platform, instance, platform user ID}; never by email or name.
type PlatformAccountLink struct {
	SchemaVersion  int                     `yaml:"schemaVersion" json:"schemaVersion"`
	ID             string                  `yaml:"id" json:"id"`
	RecordedAt     string                  `yaml:"recordedAt" json:"recordedAt"`
	Platform       string                  `yaml:"platform" json:"platform"`
	Instance       string                  `yaml:"instance" json:"instance"`
	PlatformUserID string                  `yaml:"platformUserId" json:"platformUserId"`
	LearnerID      string                  `yaml:"learnerId" json:"learnerId"`
	Confirmation   AccountLinkConfirmation `yaml:"confirmation" json:"confirmation"`
}

// AccountLinkConfirmation records who confirmed a link and when.
type AccountLinkConfirmation struct {
	ConfirmedBy string `yaml:"confirmedBy" json:"confirmedBy"`
	ConfirmedAt string `yaml:"confirmedAt" json:"confirmedAt"`
}

// LearnerConfirmation is the only accepted AccountLinkConfirmation.ConfirmedBy.
const LearnerConfirmation = "learner"

const platformAccountsDir = "platform-accounts"

// PlatformAccountLinkID is the deterministic record ID of a platform account,
// so one platform account has at most one link record per workspace.
func PlatformAccountLinkID(platform, instance, platformUserID string) string {
	sum := sha256.Sum256([]byte(platform + "\x00" + instance + "\x00" + platformUserID))
	return "pal_" + hex.EncodeToString(sum[:12])
}

// AppendPlatformAccountLink appends a learner-confirmed link record.
func (s Store) AppendPlatformAccountLink(expectedRevision string, link PlatformAccountLink) (PlatformAccountLink, error) {
	if err := s.requireRevision(expectedRevision); err != nil {
		return PlatformAccountLink{}, err
	}
	if link.Confirmation.ConfirmedBy != LearnerConfirmation {
		return PlatformAccountLink{}, fmt.Errorf("platform account link requires learner confirmation")
	}
	if link.SchemaVersion == 0 {
		link.SchemaVersion = 1
	}
	link.ID = PlatformAccountLinkID(link.Platform, link.Instance, link.PlatformUserID)
	data, err := yaml.Marshal(link)
	if err != nil {
		return PlatformAccountLink{}, fmt.Errorf("marshal platform account link %s: %w", link.ID, err)
	}
	if err := s.validate("platform-account-link.schema.json", link.ID+".yaml", data); err != nil {
		return PlatformAccountLink{}, err
	}
	if err := appendFile(filepath.Join(s.Root, platformAccountsDir, link.ID+".yaml"), data); err != nil {
		return PlatformAccountLink{}, err
	}
	return link, nil
}

// PlatformAccountLinks returns every platform account link, ordered by ID.
func (s Store) PlatformAccountLinks() ([]PlatformAccountLink, error) {
	paths, err := filepath.Glob(filepath.Join(s.Root, platformAccountsDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("list platform account links: %w", err)
	}
	sort.Strings(paths)
	links := make([]PlatformAccountLink, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if err := s.validate("platform-account-link.schema.json", filepath.Base(path), data); err != nil {
			return nil, err
		}
		var link PlatformAccountLink
		if err := yaml.Unmarshal(data, &link); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		links = append(links, link)
	}
	return links, nil
}

// LinkedAccount returns the link for a platform account, if one exists.
func (s Store) LinkedAccount(platform, instance, platformUserID string) (PlatformAccountLink, bool, error) {
	links, err := s.PlatformAccountLinks()
	if err != nil {
		return PlatformAccountLink{}, false, err
	}
	for _, link := range links {
		if link.Platform == platform && link.Instance == instance && link.PlatformUserID == platformUserID {
			return link, true, nil
		}
	}
	return PlatformAccountLink{}, false, nil
}
