package inventory

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

type Server struct {
	Name         string   `json:"name"`
	Host         string   `json:"host"`
	User         string   `json:"user,omitempty"`
	Port         int      `json:"port"`
	IdentityFile string   `json:"identity_file,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	SSHAlias     string   `json:"ssh_alias,omitempty"`
	SSHConfig    string   `json:"ssh_config,omitempty"`
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)
var hostPattern = regexp.MustCompile(`^[a-zA-Z0-9\[:][a-zA-Z0-9._:\[\]%-]*$`)
var userPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.@-]*$`)

func Validate(s Server) error {
	if !namePattern.MatchString(s.Name) {
		return fmt.Errorf("name must be 1–63 letters, numbers, dots, underscores or hyphens, starting with a letter or number")
	}
	if !hostPattern.MatchString(s.Host) || len(s.Host) > 253 {
		return fmt.Errorf("%s: host must be an IP address, hostname or SSH alias", s.Name)
	}
	if s.User != "" && (!userPattern.MatchString(s.User) || len(s.User) > 255) {
		return fmt.Errorf("%s: invalid SSH username", s.Name)
	}
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("%s: port must be between 1 and 65535", s.Name)
	}
	if (s.SSHAlias == "") != (s.SSHConfig == "") {
		return fmt.Errorf("%s: imported aliases need both ssh_alias and ssh_config", s.Name)
	}
	if s.SSHAlias != "" && !namePattern.MatchString(s.SSHAlias) {
		return fmt.Errorf("%s: invalid SSH alias", s.Name)
	}
	fields := append([]string{s.IdentityFile, s.Notes, s.SSHConfig}, s.Tags...)
	for _, field := range fields {
		if len(field) > 4096 || strings.IndexFunc(field, unicode.IsControl) >= 0 {
			return fmt.Errorf("%s: fields must be at most 4096 bytes and contain no control characters", s.Name)
		}
	}
	return nil
}

func Find(servers []Server, name string) (Server, error) {
	for _, s := range servers {
		if strings.EqualFold(s.Name, name) {
			return s, nil
		}
	}
	return Server{}, fmt.Errorf("server %q not found; use hova list to see saved servers", name)
}

func Filter(servers []Server, query, tag string) []Server {
	result := make([]Server, 0)
	query = strings.ToLower(strings.TrimSpace(query))
	for _, s := range servers {
		if tag != "" && !hasTag(s.Tags, tag) {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{s.Name, s.Host, s.User, strings.Join(s.Tags, " "), s.Notes}, " "))
		if strings.Contains(haystack, query) {
			result = append(result, s)
		}
	}
	return result
}

func hasTag(tags []string, tag string) bool {
	for _, v := range tags {
		if strings.EqualFold(v, tag) {
			return true
		}
	}
	return false
}
