package ssh

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/alzubair-alojali/hova/internal/inventory"
)

// Import reads display metadata only. OpenSSH resolves the full configuration on connection.
func Import(r io.Reader, configPath string) ([]inventory.Server, error) {
	result := []inventory.Server{}
	active := []int{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	for line := 1; scanner.Scan(); line++ {
		fields, err := splitConfig(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("SSH config line %d: %w", line, err)
		}
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		value := fields[1]
		if key == "match" {
			active = nil
			continue
		}
		if key == "host" {
			active = nil
			for _, alias := range fields[1:] {
				if strings.ContainsAny(alias, "*?!") || seen[strings.ToLower(alias)] {
					continue
				}
				s := inventory.Server{Name: alias, Host: alias, Port: 22, SSHAlias: alias, SSHConfig: configPath}
				if err := inventory.Validate(s); err != nil {
					return nil, fmt.Errorf("SSH config line %d: %w", line, err)
				}
				result = append(result, s)
				active = append(active, len(result)-1)
				seen[strings.ToLower(alias)] = true
			}
			continue
		}
		for _, index := range active {
			s := &result[index]
			switch key {
			case "hostname":
				s.Host = value
			case "user":
				s.User = value
			case "identityfile":
				if s.IdentityFile == "" {
					s.IdentityFile = value
				}
			case "port":
				n, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("SSH config line %d: invalid port", line)
				}
				s.Port = n
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, s := range result {
		if err := inventory.Validate(s); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func splitConfig(line string) ([]string, error) {
	fields := []string{}
	var token strings.Builder
	var quote rune
	escaped := false
	started := false
	for _, r := range line {
		if escaped {
			if r != '"' && r != '\'' && r != '\\' && r != ' ' {
				token.WriteRune('\\')
			}
			token.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				token.WriteRune(r)
			}
			started = true
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			started = true
			continue
		}
		if r == '#' {
			break
		}
		if r == ' ' || r == '\t' || (r == '=' && (len(fields) == 0 || (len(fields) == 1 && !started))) {
			if started {
				fields = append(fields, token.String())
				token.Reset()
				started = false
			}
			continue
		}
		token.WriteRune(r)
		started = true
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if escaped {
		token.WriteRune('\\')
	}
	if started {
		fields = append(fields, token.String())
	}
	return fields, nil
}
