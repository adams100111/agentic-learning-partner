package store

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

type SemanticConflict struct {
	Path                  string
	Reason                string
	RequiresLearnerChoice bool
}

func (c SemanticConflict) Error() string {
	if c.RequiresLearnerChoice {
		return fmt.Sprintf("%s: %s; learner resolution required", c.Path, c.Reason)
	}
	return fmt.Sprintf("%s: %s", c.Path, c.Reason)
}

func ReconcileCanonical(base, local, remote map[string][]byte) (map[string][]byte, error) {
	return reconcileSnapshots(base, local, remote)
}

func reconcileSnapshots(base, local, remote map[string][]byte) (map[string][]byte, error) {
	result := map[string][]byte{}
	paths := unionSnapshotPaths(base, local, remote)
	for _, path := range paths {
		if strings.HasPrefix(path, "state/") {
			// Derived state is regenerated after canonical reconciliation.
			continue
		}
		baseValue, baseOK := base[path]
		localValue, localOK := local[path]
		remoteValue, remoteOK := remote[path]

		switch {
		case strings.HasPrefix(path, "platform-accounts/"):
			merged, ok, err := reconcilePlatformAccountLink(path, baseValue, baseOK, localValue, localOK, remoteValue, remoteOK)
			if err != nil {
				return nil, err
			}
			if ok {
				result[path] = merged
			}
		case isAppendOnlyPath(path):
			merged, ok, err := reconcileAppendOnly(path, baseValue, baseOK, localValue, localOK, remoteValue, remoteOK)
			if err != nil {
				return nil, err
			}
			if ok {
				result[path] = merged
			}
		case path == "workspace.yaml" || path == "workspace.json":
			merged, ok, err := reconcileWorkspaceManifest(path, baseValue, baseOK, localValue, localOK, remoteValue, remoteOK)
			if err != nil {
				return nil, err
			}
			if ok {
				result[path] = merged
			}
		case strings.HasPrefix(path, "profile/") || strings.HasPrefix(path, "personas/"):
			merged, ok, err := reconcileStructuredDocument(path, baseValue, baseOK, localValue, localOK, remoteValue, remoteOK)
			if err != nil {
				return nil, err
			}
			if ok {
				result[path] = merged
			}
		default:
			merged, ok, err := reconcileOrdinary(path, baseValue, baseOK, localValue, localOK, remoteValue, remoteOK)
			if err != nil {
				return nil, err
			}
			if ok {
				result[path] = merged
			}
		}
	}
	return result, nil
}

func isAppendOnlyPath(path string) bool {
	return strings.HasPrefix(path, "evidence/") ||
		strings.HasPrefix(path, "assessments/") ||
		strings.HasPrefix(path, "sessions/") ||
		strings.HasPrefix(path, "platform-accounts/") ||
		strings.HasPrefix(path, "adaptation-decisions/") ||
		strings.HasPrefix(path, "specifications/") ||
		strings.HasPrefix(path, "authoring-plans/")
}

func reconcileAppendOnly(path string, base []byte, baseOK bool, local []byte, localOK bool, remote []byte, remoteOK bool) ([]byte, bool, error) {
	if localOK && remoteOK && bytes.Equal(local, remote) {
		return local, true, nil
	}
	if baseOK {
		if !localOK || !remoteOK {
			return nil, false, SemanticConflict{Path: path, Reason: "append-only record deletion detected"}
		}
		if bytes.Equal(local, base) {
			return remote, true, nil
		}
		if bytes.Equal(remote, base) {
			return local, true, nil
		}
		return nil, false, SemanticConflict{Path: path, Reason: "append-only record identity has different contents"}
	}
	switch {
	case localOK && !remoteOK:
		return local, true, nil
	case remoteOK && !localOK:
		return remote, true, nil
	case localOK && remoteOK:
		if bytes.Equal(local, remote) {
			return local, true, nil
		}
		return nil, false, SemanticConflict{Path: path, Reason: "same append-only identity was created with different contents"}
	default:
		return nil, false, nil
	}
}

// reconcilePlatformAccountLink reconciles a Platform Account Link record. Its
// ID is derived from {platform, instance, platform user ID}, so linking the
// same account on two devices creates the same append-only identity with
// different confirmation timestamps. Such links are equivalent when they link
// the same learner: every device keeps the earliest confirmed record, so sync
// converges. Links of one account to different learners still conflict.
func reconcilePlatformAccountLink(path string, base []byte, baseOK bool, local []byte, localOK bool, remote []byte, remoteOK bool) ([]byte, bool, error) {
	merged, ok, err := reconcileAppendOnly(path, base, baseOK, local, localOK, remote, remoteOK)
	if err == nil || baseOK || !localOK || !remoteOK {
		return merged, ok, err
	}
	localLink, parseErr := parsePlatformAccountLink(path, local)
	if parseErr != nil {
		return nil, false, parseErr
	}
	remoteLink, parseErr := parsePlatformAccountLink(path, remote)
	if parseErr != nil {
		return nil, false, parseErr
	}
	if localLink.identity() != remoteLink.identity() {
		return nil, false, SemanticConflict{Path: path, Reason: "platform account is linked to different learners", RequiresLearnerChoice: true}
	}
	switch {
	case localLink.recordedAt.Before(remoteLink.recordedAt):
		return local, true, nil
	case remoteLink.recordedAt.Before(localLink.recordedAt):
		return remote, true, nil
	case bytes.Compare(local, remote) <= 0:
		// Equal timestamps: a byte-order tiebreak keeps the choice the same
		// on every device.
		return local, true, nil
	default:
		return remote, true, nil
	}
}

type platformAccountLinkIdentity struct {
	ID             string `yaml:"id"`
	RecordedAt     string `yaml:"recordedAt"`
	Platform       string `yaml:"platform"`
	Instance       string `yaml:"instance"`
	PlatformUserID string `yaml:"platformUserId"`
	LearnerID      string `yaml:"learnerId"`
	recordedAt     time.Time
}

func (l platformAccountLinkIdentity) identity() string {
	return strings.Join([]string{l.ID, l.Platform, l.Instance, l.PlatformUserID, l.LearnerID}, "\x00")
}

func parsePlatformAccountLink(path string, data []byte) (platformAccountLinkIdentity, error) {
	var link platformAccountLinkIdentity
	if err := yaml.Unmarshal(data, &link); err != nil {
		return link, fmt.Errorf("parse platform account link %s: %w", path, err)
	}
	recordedAt, err := time.Parse(time.RFC3339, link.RecordedAt)
	if err != nil {
		return link, fmt.Errorf("parse platform account link %s recordedAt: %w", path, err)
	}
	link.recordedAt = recordedAt
	return link, nil
}

func reconcileWorkspaceManifest(path string, base []byte, baseOK bool, local []byte, localOK bool, remote []byte, remoteOK bool) ([]byte, bool, error) {
	if !localOK || !remoteOK {
		return nil, false, SemanticConflict{Path: path, Reason: "workspace manifest cannot be deleted"}
	}
	localIdentity, err := workspaceIdentity(local)
	if err != nil {
		return nil, false, err
	}
	remoteIdentity, err := workspaceIdentity(remote)
	if err != nil {
		return nil, false, err
	}
	if localIdentity != remoteIdentity {
		return nil, false, SemanticConflict{Path: path, Reason: "workspace identity differs across stores"}
	}
	return reconcileOrdinary(path, base, baseOK, local, localOK, remote, remoteOK)
}

func workspaceIdentity(data []byte) (string, error) {
	var manifest struct {
		WorkspaceID string `yaml:"workspaceId"`
		LearnerID   string `yaml:"learnerId"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parse workspace identity: %w", err)
	}
	return manifest.WorkspaceID + "\x00" + manifest.LearnerID, nil
}

func reconcileOrdinary(path string, base []byte, baseOK bool, local []byte, localOK bool, remote []byte, remoteOK bool) ([]byte, bool, error) {
	if sameOptional(local, localOK, remote, remoteOK) {
		return local, localOK, nil
	}
	if sameOptional(local, localOK, base, baseOK) {
		return remote, remoteOK, nil
	}
	if sameOptional(remote, remoteOK, base, baseOK) {
		return local, localOK, nil
	}
	return nil, false, SemanticConflict{Path: path, Reason: "both sides changed the same canonical document"}
}

func reconcileStructuredDocument(path string, base []byte, baseOK bool, local []byte, localOK bool, remote []byte, remoteOK bool) ([]byte, bool, error) {
	if sameOptional(local, localOK, remote, remoteOK) {
		return local, localOK, nil
	}
	if sameOptional(local, localOK, base, baseOK) {
		return remote, remoteOK, nil
	}
	if sameOptional(remote, remoteOK, base, baseOK) {
		return local, localOK, nil
	}
	if !localOK || !remoteOK {
		return nil, false, SemanticConflict{Path: path, Reason: "canonical profile/persona deletion conflicts with a concurrent modification", RequiresLearnerChoice: true}
	}

	var baseValue any
	if baseOK {
		if err := yaml.Unmarshal(base, &baseValue); err != nil {
			return nil, false, fmt.Errorf("parse base %s: %w", path, err)
		}
	}
	var localValue, remoteValue any
	if err := yaml.Unmarshal(local, &localValue); err != nil {
		return nil, false, fmt.Errorf("parse local %s: %w", path, err)
	}
	if err := yaml.Unmarshal(remote, &remoteValue); err != nil {
		return nil, false, fmt.Errorf("parse remote %s: %w", path, err)
	}

	merged, err := mergeSemanticValue(baseValue, localValue, remoteValue, path, localValue, remoteValue)
	if err != nil {
		return nil, false, err
	}
	output, err := yaml.Marshal(merged)
	if err != nil {
		return nil, false, fmt.Errorf("encode reconciled %s: %w", path, err)
	}
	return output, true, nil
}

func mergeSemanticValue(base, local, remote any, path string, localContext, remoteContext any) (any, error) {
	if reflect.DeepEqual(local, remote) {
		return local, nil
	}
	if reflect.DeepEqual(local, base) {
		return remote, nil
	}
	if reflect.DeepEqual(remote, base) {
		return local, nil
	}

	localMap, localIsMap := local.(map[string]any)
	remoteMap, remoteIsMap := remote.(map[string]any)
	baseMap, _ := base.(map[string]any)
	if localIsMap && remoteIsMap {
		return mergeSemanticMap(baseMap, localMap, remoteMap, path, chooseContext(localMap, localContext), chooseContext(remoteMap, remoteContext))
	}

	if strings.HasSuffix(path, "/goals") {
		localArray, localOK := local.([]any)
		remoteArray, remoteOK := remote.([]any)
		baseArray, _ := base.([]any)
		if localOK && remoteOK {
			return mergeIdentifiedArray(baseArray, localArray, remoteArray, path)
		}
	}

	localScore := provenanceAuthority(localContext)
	remoteScore := provenanceAuthority(remoteContext)
	if localScore > remoteScore {
		return local, nil
	}
	if remoteScore > localScore {
		return remote, nil
	}

	requiresLearner := localScore >= 4 && remoteScore >= 4
	return nil, SemanticConflict{
		Path:                  path,
		Reason:                "both sides changed the same semantic field with equal authority",
		RequiresLearnerChoice: requiresLearner,
	}
}

func mergeSemanticMap(base, local, remote map[string]any, path string, localContext, remoteContext any) (map[string]any, error) {
	result := map[string]any{}
	keys := map[string]struct{}{}
	for key := range base {
		keys[key] = struct{}{}
	}
	for key := range local {
		keys[key] = struct{}{}
	}
	for key := range remote {
		keys[key] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	for _, key := range sorted {
		baseValue, baseOK := base[key]
		localValue, localOK := local[key]
		remoteValue, remoteOK := remote[key]
		childPath := path + "/" + key

		if !localOK || !remoteOK {
			switch {
			case !baseOK && localOK:
				result[key] = localValue
				continue
			case !baseOK && remoteOK:
				result[key] = remoteValue
				continue
			case baseOK && !localOK && remoteOK && reflect.DeepEqual(remoteValue, baseValue):
				continue
			case baseOK && !remoteOK && localOK && reflect.DeepEqual(localValue, baseValue):
				continue
			default:
				return nil, SemanticConflict{Path: childPath, Reason: "field deletion conflicts with a concurrent change", RequiresLearnerChoice: provenanceAuthority(localContext) >= 4 && provenanceAuthority(remoteContext) >= 4}
			}
		}

		merged, err := mergeSemanticValue(baseValue, localValue, remoteValue, childPath, chooseContext(localMapContext(localValue), localContext), chooseContext(localMapContext(remoteValue), remoteContext))
		if err != nil {
			return nil, err
		}
		result[key] = merged
	}
	return result, nil
}

func mergeIdentifiedArray(base, local, remote []any, path string) ([]any, error) {
	baseByID := identifiedItems(base)
	localByID := identifiedItems(local)
	remoteByID := identifiedItems(remote)
	if localByID == nil || remoteByID == nil || (len(base) > 0 && baseByID == nil) {
		return nil, SemanticConflict{Path: path, Reason: "concurrent list changes cannot be merged safely"}
	}

	keys := map[string]struct{}{}
	for key := range baseByID {
		keys[key] = struct{}{}
	}
	for key := range localByID {
		keys[key] = struct{}{}
	}
	for key := range remoteByID {
		keys[key] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)

	result := make([]any, 0, len(sorted))
	for _, id := range sorted {
		b, bOK := baseByID[id]
		l, lOK := localByID[id]
		r, rOK := remoteByID[id]
		if !lOK || !rOK {
			switch {
			case !bOK && lOK:
				result = append(result, l)
				continue
			case !bOK && rOK:
				result = append(result, r)
				continue
			case bOK && !lOK && rOK && reflect.DeepEqual(r, b):
				continue
			case bOK && !rOK && lOK && reflect.DeepEqual(l, b):
				continue
			default:
				return nil, SemanticConflict{Path: path + "/" + id, Reason: "identified claim deletion conflicts with a concurrent change", RequiresLearnerChoice: true}
			}
		}
		merged, err := mergeSemanticValue(b, l, r, path+"/"+id, l, r)
		if err != nil {
			return nil, err
		}
		result = append(result, merged)
	}
	return result, nil
}

func identifiedItems(values []any) map[string]any {
	result := map[string]any{}
	for _, value := range values {
		item, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		id, ok := item["id"].(string)
		if !ok || id == "" {
			return nil
		}
		if _, duplicate := result[id]; duplicate {
			return nil
		}
		result[id] = item
	}
	return result
}

func chooseContext(candidate any, fallback any) any {
	if provenanceAuthority(candidate) >= 0 {
		return candidate
	}
	return fallback
}

func localMapContext(value any) any {
	if mapping, ok := value.(map[string]any); ok {
		return mapping
	}
	return nil
}

func provenanceAuthority(value any) int {
	mapping, ok := value.(map[string]any)
	if !ok {
		return -1
	}
	raw, exists := mapping["provenance"]
	if !exists {
		return -1
	}
	switch provenance := raw.(type) {
	case map[string]any:
		return oneProvenanceAuthority(provenance)
	case []any:
		score := -1
		for _, entry := range provenance {
			if candidate, ok := entry.(map[string]any); ok {
				if current := oneProvenanceAuthority(candidate); current > score {
					score = current
				}
			}
		}
		return score
	default:
		return -1
	}
}

func oneProvenanceAuthority(provenance map[string]any) int {
	if intent, _ := provenance["intent"].(string); intent == "correction" {
		return 5
	}
	switch source, _ := provenance["sourceType"].(string); source {
	case "learner-stated":
		return 4
	case "observed", "evaluated":
		return 3
	case "imported":
		return 2
	case "agent-inferred":
		return 1
	case "agent-proposed":
		return 0
	default:
		return -1
	}
}

func unionSnapshotPaths(snapshots ...map[string][]byte) []string {
	seen := map[string]struct{}{}
	for _, snapshot := range snapshots {
		for path := range snapshot {
			seen[path] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func sameOptional(left []byte, leftOK bool, right []byte, rightOK bool) bool {
	return leftOK == rightOK && (!leftOK || bytes.Equal(left, right))
}
