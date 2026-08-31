package taiga

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type ItemRef struct {
	Project string `json:"project"`
	Ref     int    `json:"ref"`
}

func ParseItemRef(value, defaultProject string) (ItemRef, error) {
	return parseTypedItemRef(value, defaultProject, map[string]struct{}{"issue": {}})
}

func ParseStoryRef(value, defaultProject string) (ItemRef, error) {
	return parseTypedItemRef(value, defaultProject, map[string]struct{}{"us": {}, "story": {}, "userstory": {}, "user-story": {}})
}

func ParseTaskRef(value, defaultProject string) (ItemRef, error) {
	return parseTypedItemRef(value, defaultProject, map[string]struct{}{"task": {}})
}

func parseTypedItemRef(value, defaultProject string, kinds map[string]struct{}) (ItemRef, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ItemRef{}, fmt.Errorf("item reference cannot be empty")
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parts := splitPath(parsed.Path)
		for index := 0; index+3 < len(parts); index++ {
			_, kindMatches := kinds[parts[index+2]]
			if parts[index] == "project" && kindMatches {
				ref, err := strconv.Atoi(parts[index+3])
				if err != nil || ref <= 0 {
					return ItemRef{}, fmt.Errorf("invalid issue ref in URL %q", value)
				}
				return ItemRef{Project: parts[index+1], Ref: ref}, nil
			}
		}
		return ItemRef{}, fmt.Errorf("URL %q is not a supported Taiga item URL", value)
	}
	if project, rawRef, ok := strings.Cut(value, "#"); ok {
		ref, err := strconv.Atoi(rawRef)
		if strings.TrimSpace(project) == "" || err != nil || ref <= 0 {
			return ItemRef{}, fmt.Errorf("invalid issue reference %q", value)
		}
		return ItemRef{Project: strings.TrimSpace(project), Ref: ref}, nil
	}
	ref, err := strconv.Atoi(value)
	if err != nil || ref <= 0 {
		return ItemRef{}, fmt.Errorf("invalid issue reference %q", value)
	}
	if strings.TrimSpace(defaultProject) == "" {
		return ItemRef{}, fmt.Errorf("no project selected; run `taiga project use <slug>` or pass --project")
	}
	return ItemRef{Project: strings.TrimSpace(defaultProject), Ref: ref}, nil
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	result := parts[:0]
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
