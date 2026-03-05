package sqlc

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	BaseDir, _ = os.Getwd()

	SqlcDir    = filepath.Join(BaseDir, "gen", "sqlc")
	ModelsFile = filepath.Join(SqlcDir, "models.go")

	// Combined pattern to find existing marshal/unmarshal methods in one pass.
	patternMethod   = regexp.MustCompile(`func \(e \*?([A-Za-z]+)\) (?:Un)?MarshalText`)
	patternEnumType = regexp.MustCompile(`^type ([A-Za-z]+) string$`)
	patternConstVal = regexp.MustCompile(`^\t+([A-Za-z][A-Za-z0-9_]*)\s+(\w+)\s*=\s*"(.+)"$`)
)

func findExistingMethods(content []byte) map[string]bool {
	methodsExist := make(map[string]bool)
	for _, match := range patternMethod.FindAllSubmatch(content, -1) {
		if len(match) > 1 {
			methodsExist[string(match[1])] = true
		}
	}
	return methodsExist
}

// generateMethods writes marshal/unmarshal methods directly into buf.
func generateMethods(buf *bytes.Buffer, enumType string, values []string) {
	if len(values) == 0 {
		return
	}

	lowerName := strings.ToLower(enumType[:1]) + enumType[1:]
	cases := strings.Join(values, ", ")

	fmt.Fprintf(buf, "\nfunc (e *%s) UnmarshalText(b []byte) error {\n", enumType)
	fmt.Fprintf(buf, "\tv := %s(b)\n", enumType)
	buf.WriteString("\tswitch v {\n")
	fmt.Fprintf(buf, "\tcase %s:\n", cases)
	buf.WriteString("\t\t*e = v\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(buf, "\treturn fmt.Errorf(\"invalid %s %%q\", string(b))\n}\n", lowerName)
	fmt.Fprintf(buf, "\nfunc (e %s) MarshalText() ([]byte, error) { return []byte(e), nil }\n", enumType)
}

func TransformModels() (bool, int) {
	fmt.Printf("Transforming models ...\n")
	startTime := time.Now()

	content, err := os.ReadFile(ModelsFile)
	if err != nil {
		fmt.Printf("Failed to read models file %v\n", err)
		return false, 0
	}

	methodsExist := findExistingMethods(content)
	fmt.Printf("Found %d existing marshal methods\n", len(methodsExist))

	output := bytes.NewBuffer(make([]byte, 0, len(content)+512))
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var (
		enumType         string
		values           []string
		inConst          bool
		methodsGenerated int
	)

	for scanner.Scan() {
		line := scanner.Text()

		if match := patternEnumType.FindStringSubmatch(line); match != nil {
			enumType = match[1]
			inConst = false
			values = values[:0] // reuse slice
			output.WriteString(line)
			output.WriteByte('\n')
			continue
		}

		if inConst {
			if strings.TrimSpace(line) == ")" {
				output.WriteString(line)
				output.WriteByte('\n')

				if enumType != "" && !methodsExist[enumType] && len(values) > 0 {
					generateMethods(output, enumType, values)
					methodsGenerated++
				}

				inConst = false
				enumType = ""
				values = values[:0]
				continue
			}

			if match := patternConstVal.FindStringSubmatch(line); match != nil && match[2] == enumType {
				values = append(values, match[1])
			}
		} else if strings.TrimSpace(line) == "const (" && enumType != "" {
			inConst = true
			values = values[:0]
		}

		output.WriteString(line)
		output.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Failed to scan models file: %v\n", err)
		return false, 0
	}

	if err := os.WriteFile(ModelsFile, output.Bytes(), 0600); err != nil {
		fmt.Printf("Failed to write models file %v\n", err)
		return false, 0
	}

	fmt.Printf("Model transformation %v\n", time.Since(startTime))
	fmt.Printf("Generated %d marshal method(s)\n", methodsGenerated)
	return true, methodsGenerated
}
