package server

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var stepPattern = regexp.MustCompile(`@step:([A-Za-z0-9._-]+)`)

func sourceDocument(language string) (SourceDocument, error) {
	root, err := LabRoot()
	if err != nil {
		return SourceDocument{}, err
	}
	files, err := languageSourceFiles(root, language)
	if err != nil {
		return SourceDocument{}, err
	}
	if len(files) == 0 {
		return SourceDocument{}, fmt.Errorf("no source file found for language %q", language)
	}

	var content strings.Builder
	anchors := map[string]int{}
	lineOffset := 0
	for index, path := range files {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return SourceDocument{}, readErr
		}
		if index > 0 {
			content.WriteString("\n")
			lineOffset++
		}
		fileContent := string(data)
		content.WriteString(fileContent)
		if !strings.HasSuffix(fileContent, "\n") {
			content.WriteByte('\n')
		}
		scanner := bufio.NewScanner(strings.NewReader(fileContent))
		line := 0
		for scanner.Scan() {
			line++
			if match := stepPattern.FindStringSubmatch(scanner.Text()); len(match) == 2 {
				anchors[match[1]] = lineOffset + line + 1
			}
		}
		lineOffset += line
	}
	relative := labRelativePath(files[0])
	if len(files) > 1 {
		relative = filepath.ToSlash(filepath.Dir(relative))
	}
	return SourceDocument{Language: language, Path: relative, Content: content.String(), Anchors: anchors}, nil
}

func languageSourceFiles(root, language string) ([]string, error) {
	var files []string
	switch language {
	case LanguageGo:
		files = []string{filepath.Join(root, "server", "algorithms.go")}
	case LanguagePython:
		files = []string{filepath.Join(root, "runners", "python", "algorithms.py")}
	case LanguageJava:
		files = []string{filepath.Join(root, "runners", "java", "RateLimiterRunner.java")}
	case LanguageJavaScript:
		files = []string{filepath.Join(root, "runners", "js", "algorithms.mjs")}
	default:
		return nil, fmt.Errorf("unsupported language %q", language)
	}
	for _, path := range files {
		if _, err := os.Stat(path); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func enrichSource(response *RunResponse, language string) error {
	document, err := sourceDocument(language)
	if err != nil {
		return err
	}
	response.Source = document
	response.Language = language
	for index := range response.Events {
		if response.Events[index].Source.Path == "" {
			response.Events[index].Source.Path = document.Path
		}
		if response.Events[index].Source.Line <= 0 {
			response.Events[index].Source.Line = document.Anchors[response.Events[index].StepID]
		}
	}
	return nil
}
