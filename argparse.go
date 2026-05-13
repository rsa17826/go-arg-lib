package argparse

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

type ArgumentData struct {
	Keys        []string
	AfterCount  int
	Target      any
	Description string
	VarArgs     bool
	AllowDupes  bool
}

type ParseOptions struct {
	DisableDefaultHelp bool
}

func ParseArgs(argDefinitions []ArgumentData, opts ...ParseOptions) {
	args := os.Args[1:]
	disableHelp := false
	if len(opts) > 0 && opts[0].DisableDefaultHelp {
		disableHelp = true
	}

	// 1. Handle "--" separator
	for i, v := range args {
		if v == "--" {
			args = args[:i]
			break
		}
	}

	// 2. Automatic Help Logic
	if !disableHelp {
		for i, arg := range args {
			if matches([]string{"h", "help"}, arg) {
				filter := args[i+1:]
				PrintHelp(argDefinitions, filter)
				os.Exit(0)
			}
		}
	}

	// 3. Parsing Logic
	for i := 0; i < len(args); {
		found := false
		for _, def := range argDefinitions {
			if !matches(def.Keys, args[i]) {
				continue
			}
			found = true

			if def.VarArgs {
				// Collect at least AfterCount values, then keep consuming
				// tokens that don't look like a registered flag.
				values, newI := collectVarArgs(args, i+1, def.AfterCount, argDefinitions)
				if def.AllowDupes {
					// Append this invocation's values to whatever was there before.
					if t, ok := def.Target.(*[]string); ok {
						*t = append(*t, values...)
					}
				} else {
					if t, ok := def.Target.(*[]string); ok {
						*t = values
					}
				}
				i = newI
			} else if def.AllowDupes {
				if def.AfterCount == 0 {
					// No value expected — count occurrences.
					if t, ok := def.Target.(*int); ok {
						*t++
					}
					i++
				} else if def.AfterCount == 1 && i+1 < len(args) {
					if t, ok := def.Target.(*[]string); ok {
						*t = append(*t, args[i+1])
					}
					i += 2
				} else if def.AfterCount > 1 && i+def.AfterCount < len(args) {
					if t, ok := def.Target.(*[]string); ok {
						*t = append(*t, args[i+1:i+1+def.AfterCount]...)
					}
					i += 1 + def.AfterCount
				} else {
					i++
				}

			} else {
				// Original single-use logic.
				if def.AfterCount == 0 {
					if t, ok := def.Target.(*bool); ok {
						*t = true
					}
					i++
				} else if def.AfterCount == 1 && i+1 < len(args) {
					if t, ok := def.Target.(*string); ok {
						*t = args[i+1]
					}
					i += 2
				} else if def.AfterCount > 1 && i+def.AfterCount < len(args) {
					if t, ok := def.Target.(*[]string); ok {
						*t = args[i+1 : i+1+def.AfterCount]
					}
					i += 1 + def.AfterCount
				} else {
					i++
				}
			}
			break
		}
		if !found {
			i++
		}
	}
}

// collectVarArgs collects at least minCount tokens starting at args[start],
// then continues collecting as long as the next token is not a registered key.
// Returns the collected slice and the new index into args.
func collectVarArgs(args []string, start, minCount int, allDefs []ArgumentData) ([]string, int) {
	collected := []string{}
	i := start

	// Consume the minimum required values regardless of whether they look like flags.
	for j := 0; j < minCount && i < len(args); j++ {
		collected = append(collected, args[i])
		i++
	}

	// Consume additional tokens until we hit something that matches a known key.
	for i < len(args) && !isAnyKey(allDefs, args[i]) {
		collected = append(collected, args[i])
		i++
	}

	return collected, i
}

// isAnyKey reports whether token matches a key in any of the provided definitions.
func isAnyKey(defs []ArgumentData, token string) bool {
	for _, def := range defs {
		if matches(def.Keys, token) {
			return true
		}
	}
	return false
}

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

func PrintHelp(defs []ArgumentData, filters []string) {
	fmt.Println("Usage Options:")
	fmt.Printf("  %-25s %-15s %s\n", "FLAGS", "EXPECTS", "DESCRIPTION")
	fmt.Printf("  %-25s %-15s %s\n", "-----", "-------", "-----------")

	for _, def := range defs {
		// Format keys: e.g., "-h, --help"
		keysStr := ""
		for i, k := range def.Keys {
			prefix := "-"
			if len(k) > 1 {
				prefix = "--"
			}
			keysStr += prefix + k
			if i < len(def.Keys)-1 {
				keysStr += ", "
			}
		}

		// Determine the "Expects" hint based on AfterCount and VarArgs
		expects := ""
		if def.VarArgs {
			expects = Yellow + "<val1>...<valN>" + Reset
		} else if def.AfterCount == 0 {
			expects = Gray + "[flag]" + Reset
		} else if def.AfterCount == 1 {
			expects = Yellow + "<value>" + Reset
		} else {
			expects = fmt.Sprintf("%s<%d values>%s", Yellow, def.AfterCount, Reset)
		}

		// Filtering logic
		if len(filters) > 0 {
			match := false
			for _, f := range filters {
				if strings.Contains(keysStr, f) || strings.Contains(strings.ToLower(def.Description), strings.ToLower(f)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Print main row
		fmt.Printf("  %-38s %-25s %s\n", keysStr, expects, def.Description)

		// 4. Print examples (Gray/Italic style)
		if def.AllowDupes || def.VarArgs || def.AfterCount > 1 {
			example := fmt.Sprintf("    %sExample: %s%s", Gray, formatExample(def), Reset)
			fmt.Println(example)
		}
	}
}

// formatExample generates a dummy usage string based on the definition
func formatExample(def ArgumentData) string {
	primaryKey := "-" + def.Keys[0]
	if len(def.Keys[0]) > 1 {
		primaryKey = "-" + primaryKey
	}

	if def.VarArgs {
		if def.AllowDupes {
			return fmt.Sprintf("%s item1 item2 item3 or %s item1 item2 %s item3", primaryKey, primaryKey, primaryKey)
		}
		return fmt.Sprintf("%s item1 item2 item3", primaryKey)
	}
	if def.AllowDupes && def.AfterCount == 0 {
		return fmt.Sprintf("%s %s %s (counts occurrences)", primaryKey, primaryKey, primaryKey)
	}
	if def.AllowDupes && def.AfterCount == 1 {
		return fmt.Sprintf("%s val1 %s val2", primaryKey, primaryKey)
	}
	if def.AfterCount > 1 {
		vals := []string{}
		for i := 1; i <= def.AfterCount; i++ {
			vals = append(vals, fmt.Sprintf("val%d", i))
		}
		return fmt.Sprintf("%s %s", primaryKey, strings.Join(vals, " "))
	}
	return ""
}

func matches(keys []string, input string) bool {
	clean := strings.TrimLeft(input, "-")
	return slices.Contains(keys, clean)
}
