package hosts

import (
	"net"
	"regexp"
	"strings"
)

type LineType int

const (
	LineEntry LineType = iota
	LineDisabled
	LineComment
)

type Line struct {
	Type      LineType
	IP        string
	Hostnames []string
	Raw       string
}

var disabledRe = regexp.MustCompile(`^#\s*([\d.]+|[0-9a-fA-F:]+)\s+(.+)$`)

func Parse(content string) []Line {
	var lines []Line
	for _, raw := range splitLines(content) {
		lines = append(lines, parseLine(raw))
	}
	return lines
}

func splitLines(content string) []string {
	parts := strings.Split(content, "\n")
	var lines []string
	for i, p := range parts {
		if i < len(parts)-1 {
			lines = append(lines, p+"\n")
		} else if p != "" {
			lines = append(lines, p)
		}
	}
	return lines
}

func parseLine(raw string) Line {
	trimmed := strings.TrimRight(raw, "\n")

	if m := disabledRe.FindStringSubmatch(trimmed); m != nil {
		ip := m[1]
		if net.ParseIP(ip) != nil {
			hostnames := strings.Fields(m[2])
			return Line{Type: LineDisabled, IP: ip, Hostnames: hostnames, Raw: raw}
		}
	}

	if !strings.HasPrefix(trimmed, "#") && trimmed != "" {
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 && net.ParseIP(fields[0]) != nil {
			return Line{Type: LineEntry, IP: fields[0], Hostnames: fields[1:], Raw: raw}
		}
	}

	return Line{Type: LineComment, Raw: raw}
}

func Format(lines []Line) string {
	var sb strings.Builder
	for _, l := range lines {
		switch l.Type {
		case LineComment:
			sb.WriteString(l.Raw)
		case LineEntry:
			sb.WriteString(l.IP)
			for _, h := range l.Hostnames {
				sb.WriteByte(' ')
				sb.WriteString(h)
			}
			sb.WriteByte('\n')
		case LineDisabled:
			sb.WriteString("# ")
			sb.WriteString(l.IP)
			for _, h := range l.Hostnames {
				sb.WriteByte(' ')
				sb.WriteString(h)
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
