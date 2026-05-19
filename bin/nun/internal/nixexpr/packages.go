package nixexpr

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type PackageSetFile struct {
	Name     string
	Path     string
	Packages []string
	List     ListRange
	Source   string
}

type ListRange struct {
	OpenLine  int
	CloseLine int
	Indent    string
}

var identifierRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ValidIdentifier(value string) bool {
	return identifierRE.MatchString(value)
}

func ReadPackageSetNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".nix" {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".nix"))
	}
	sort.Strings(names)
	return names, nil
}

func ReadPackageSetFiles(dir string) ([]PackageSetFile, error) {
	names, err := ReadPackageSetNames(dir)
	if err != nil {
		return nil, err
	}
	files := make([]PackageSetFile, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name+".nix")
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		file, err := ParsePackageSet(name, path, string(source))
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func ParsePackageSet(name, path, source string) (PackageSetFile, error) {
	packages, list, err := ParseTopLevelList(source)
	if err != nil {
		return PackageSetFile{}, fmt.Errorf("%s: %w", path, err)
	}
	return PackageSetFile{
		Name:     name,
		Path:     path,
		Packages: packages,
		List:     list,
		Source:   source,
	}, nil
}

func ParseTopLevelList(source string) ([]string, ListRange, error) {
	lines := strings.Split(source, "\n")
	openLine := -1
	closeLine := -1
	depth := 0
	var packages []string
	indent := "  "

	for i, raw := range lines {
		clean := stripLineComment(raw)
		for _, r := range clean {
			switch r {
			case '[':
				if depth == 0 && openLine == -1 {
					openLine = i
				}
				depth++
			case ']':
				depth--
				if depth == 0 {
					closeLine = i
				}
			}
		}
		if openLine >= 0 && closeLine == -1 && i > openLine {
			value := strings.TrimSpace(stripLineComment(raw))
			value = strings.TrimSuffix(value, ",")
			if value != "" && !strings.ContainsAny(value, "[];=") {
				packages = append(packages, value)
				indent = leadingWhitespace(raw)
			}
		}
		if closeLine >= 0 {
			break
		}
	}
	if openLine < 0 || closeLine < 0 {
		return nil, ListRange{}, fmt.Errorf("could not find a complete list expression")
	}
	return packages, ListRange{OpenLine: openLine, CloseLine: closeLine, Indent: indent}, nil
}

func AddListItem(source, item string) (string, error) {
	items, list, err := ParseTopLevelList(source)
	if err != nil {
		return "", err
	}
	for _, existing := range items {
		if existing == item {
			return source, nil
		}
	}
	lines := strings.Split(source, "\n")
	insert := list.Indent + item
	next := append([]string{}, lines[:list.CloseLine]...)
	next = append(next, insert)
	next = append(next, lines[list.CloseLine:]...)
	return strings.Join(next, "\n"), nil
}

func AddNamedListItem(source, listName, item string) (string, error) {
	items, list, err := ParseNamedList(source, listName)
	if err != nil {
		return "", err
	}
	for _, existing := range items {
		if existing == item {
			return source, nil
		}
	}
	lines := strings.Split(source, "\n")
	insert := list.Indent + item
	next := append([]string{}, lines[:list.CloseLine]...)
	next = append(next, insert)
	next = append(next, lines[list.CloseLine:]...)
	return strings.Join(next, "\n"), nil
}

func AddListItemAfter(source, marker, item string) (string, error) {
	idx := strings.Index(source, marker)
	if idx < 0 {
		return "", fmt.Errorf("could not find marker %q", marker)
	}
	prefix := source[:idx]
	linesBefore := strings.Count(prefix, "\n")
	lines := strings.Split(source, "\n")
	items, list, err := parseListFromLine(lines, linesBefore)
	if err != nil {
		return "", err
	}
	for _, existing := range items {
		if existing == item {
			return source, nil
		}
	}
	insert := list.Indent + item
	next := append([]string{}, lines[:list.CloseLine]...)
	next = append(next, insert)
	next = append(next, lines[list.CloseLine:]...)
	return strings.Join(next, "\n"), nil
}

func ParseNamedList(source, listName string) ([]string, ListRange, error) {
	lines := strings.Split(source, "\n")
	for i, raw := range lines {
		clean := strings.TrimSpace(stripLineComment(raw))
		if !strings.HasPrefix(clean, listName+" =") {
			continue
		}
		packages, list, err := parseListFromLine(lines, i)
		if err != nil {
			return nil, ListRange{}, fmt.Errorf("%s: %w", listName, err)
		}
		return packages, list, nil
	}
	return nil, ListRange{}, fmt.Errorf("could not find list %q", listName)
}

func parseListFromLine(lines []string, start int) ([]string, ListRange, error) {
	openLine := -1
	closeLine := -1
	depth := 0
	var packages []string
	indent := "  "
	for i := start; i < len(lines); i++ {
		raw := lines[i]
		clean := stripLineComment(raw)
		for _, r := range clean {
			switch r {
			case '[':
				if depth == 0 && openLine == -1 {
					openLine = i
				}
				depth++
			case ']':
				depth--
				if depth == 0 {
					closeLine = i
				}
			}
		}
		if openLine >= 0 && closeLine == -1 && i > openLine {
			value := strings.TrimSpace(stripLineComment(raw))
			value = strings.TrimSuffix(value, ",")
			if value != "" && !strings.ContainsAny(value, "[];=") {
				packages = append(packages, value)
				indent = leadingWhitespace(raw)
			}
		}
		if closeLine >= 0 {
			return packages, ListRange{OpenLine: openLine, CloseLine: closeLine, Indent: indent}, nil
		}
	}
	return nil, ListRange{}, fmt.Errorf("could not find a complete list expression")
}

func RemoveListItem(source, item string) (string, error) {
	_, list, err := ParseTopLevelList(source)
	if err != nil {
		return "", err
	}
	lines := strings.Split(source, "\n")
	next := make([]string, 0, len(lines))
	for i, line := range lines {
		if i > list.OpenLine && i < list.CloseLine {
			clean := strings.TrimSpace(stripLineComment(line))
			clean = strings.TrimSuffix(clean, ",")
			if clean == item {
				continue
			}
		}
		next = append(next, line)
	}
	return strings.Join(next, "\n"), nil
}

func RemoveNamedListItem(source, listName, item string) (string, error) {
	_, list, err := ParseNamedList(source, listName)
	if err != nil {
		return "", err
	}
	lines := strings.Split(source, "\n")
	next := make([]string, 0, len(lines))
	for i, line := range lines {
		if i > list.OpenLine && i < list.CloseLine {
			clean := strings.TrimSpace(stripLineComment(line))
			clean = strings.TrimSuffix(clean, ",")
			// For quoted strings (like homebrew packages), compare without quotes
			if strings.HasPrefix(clean, `"`) && strings.HasSuffix(clean, `"`) {
				clean = clean[1 : len(clean)-1]
			}
			if clean == item {
				continue
			}
		}
		next = append(next, line)
	}
	return strings.Join(next, "\n"), nil
}

func RemoveListItemAfter(source, marker, item string) (string, error) {
	idx := strings.Index(source, marker)
	if idx < 0 {
		return "", fmt.Errorf("could not find marker %q", marker)
	}
	prefix := source[:idx]
	linesBefore := strings.Count(prefix, "\n")
	lines := strings.Split(source, "\n")
	_, list, err := parseListFromLine(lines, linesBefore)
	if err != nil {
		return "", err
	}
	next := make([]string, 0, len(lines))
	for i, line := range lines {
		if i > list.OpenLine && i < list.CloseLine {
			clean := strings.TrimSpace(stripLineComment(line))
			clean = strings.TrimSuffix(clean, ",")
			if clean == item {
				continue
			}
		}
		next = append(next, line)
	}
	return strings.Join(next, "\n"), nil
}

func stripLineComment(line string) string {
	if idx := strings.IndexRune(line, '#'); idx >= 0 {
		return line[:idx]
	}
	return line
}

func leadingWhitespace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}
