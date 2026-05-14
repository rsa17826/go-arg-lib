package argparse

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
)

type ArgumentData struct {
	Keys        []string
	AfterCount  int
	Target      any
	Description string
	VarArgs     bool
	AllowDupes  bool
	ExampleArgs []string
	Default     []any
}

var (
	ParseArgsData = make(map[string][]ArgumentData)
	mu            sync.RWMutex
	done          = make(chan struct{}) // The blocking signal
)
var helpArgs []string

func ParseArgs(argData []ArgumentData) {
	_ParseArgs(argData, getCallerPackageName())
	<-done
}
func _ParseArgs(argData []ArgumentData, caller string) {
	mu.Lock()
	defer mu.Unlock()
	ParseArgsData[caller] = argData
}

func assign(target any, value any) {
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return
	}

	targetElem := targetVal.Elem()
	val := reflect.ValueOf(value)

	if val.Type().AssignableTo(targetElem.Type()) {
		targetElem.Set(val)
		return
	}

	if targetElem.Kind() == reflect.Slice && (val.Kind() == reflect.Slice || val.Kind() == reflect.Array) {
		newSlice := reflect.MakeSlice(targetElem.Type(), val.Len(), val.Len())
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)

			if item.Kind() == reflect.Interface {
				item = item.Elem()
			}

			targetType := targetElem.Type().Elem()
			if item.Type().ConvertibleTo(targetType) {
				newSlice.Index(i).Set(item.Convert(targetType))
			}
		}
		targetElem.Set(newSlice)
		return
	}

	if val.Type().ConvertibleTo(targetElem.Type()) {
		targetElem.Set(val.Convert(targetElem.Type()))
	} else {
		println("failed to set default for", target, value)
	}
}

func getCallerPackageName() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}

	details := runtime.FuncForPC(pc)
	if details == nil {
		return "unknown"
	}

	fullName := details.Name()

	lastDot := strings.LastIndex(fullName, ".")
	if lastDot == -1 {
		return fullName
	}

	pkgPath := fullName[:lastDot]
	return pkgPath
}

func init() {
	_ParseArgs([]ArgumentData{{
		Keys:        []string{"help", "h"},
		AfterCount:  0,
		Target:      &helpArgs,
		Description: "show help; optionally filter by keyword",
		VarArgs:     true,
		AllowDupes:  false,
	}}, getCallerPackageName())
	println("argparse1")
	go func() {
		println("argparse2")
		runtime.Gosched()
		println("argparse3")
		mu.RLock()
		args := os.Args[1:]

		for i, v := range args {
			if v == "--" {
				args = args[:i]
				break
			}
		}

		for _, argSlice := range ParseArgsData {
			for i := range argSlice {
				def := &argSlice[i]
				if len(def.Default) == 0 {
					continue
				}
				targetKind := reflect.TypeOf(def.Target).Elem().Kind()
				if targetKind != reflect.Slice {
					assign(def.Target, def.Default[0])
				} else {
					assign(def.Target, def.Default)
				}
			}
		}

		for i := 0; i < len(args); {
			found := false
		argLoop:
			for _, argSlice := range ParseArgsData {
				for _, def := range argSlice {
					if !matches(def.Keys, args[i]) {
						continue
					}
					found = true

					if def.VarArgs {
						values, newI := collectVarArgs(args, i+1, def.AfterCount, ParseArgsData["MAIN"])
						if def.AllowDupes {
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
					break argLoop
				}
			}
			if !found {
				i++
			}
		}

		if helpArgs != nil {
			PrintHelp(helpArgs)
			os.Exit(0)
		}
		mu.RUnlock()

		// Signal everyone that parsing is finished
		close(done)
	}()
}

func collectVarArgs(args []string, start, minCount int, allDefs []ArgumentData) ([]string, int) {
	collected := []string{}
	i := start

	for j := 0; j < minCount && i < len(args); j++ {
		collected = append(collected, args[i])
		i++
	}

	for i < len(args) && !isAnyKey(allDefs, args[i]) {
		collected = append(collected, args[i])
		i++
	}

	return collected, i
}

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

func PrintHelp(filters []string) {
	const (
		col1Width = 20
		col2Width = 30
	)

	fmt.Printf("  %-*s %-*s %s\n", col1Width, "FLAGS", col2Width, "EXPECTS", "DESCRIPTION")
	fmt.Printf("  %s%s\n%s", Gray, strings.Repeat("-", col1Width+col2Width+13+20), Reset)
	parseThisData := func(defs []ArgumentData) {
		for _, def := range defs {
			keysFormatted := []string{}
			rawKeysLen := 0
			for i, k := range def.Keys {
				p := "-"
				if len(k) > 1 {
					p = "--"
				}
				keysFormatted = append(keysFormatted, Cyan+p+k+Reset)
				rawKeysLen += len(p) + len(k)
				if i < len(def.Keys)-1 {
					rawKeysLen += 2
				}
			}
			keysStr := strings.Join(keysFormatted, ", ")

			var expects, rawExpects string
			if def.VarArgs {
				expects, rawExpects = Yellow+"<val1>...<valN>"+Reset, "<val1>...<valN>"
			} else if def.AfterCount == 0 {
				expects, rawExpects = Blue+"[flag]"+Reset, "[flag]"
			} else {
				noDataCounter := 0
				for i := range def.AfterCount {
					name := "value"
					var default_ any = nil
					var dataFound = false
					if i > 0 {
						expects += " "
						rawExpects += " "
					}
					if i < len(def.ExampleArgs) {
						dataFound = true
						if noDataCounter > 0 {
							if noDataCounter == 1 {
								expects += Yellow + "<1 value>" + Reset
								rawExpects += "<1 value>"
							} else {
								expects += fmt.Sprintf("%s<%d values>%s", Yellow, noDataCounter, Reset)
								rawExpects += fmt.Sprintf("<%d values>", noDataCounter)
								noDataCounter = 0
							}
						}
						name = def.ExampleArgs[i]
					}
					if i < len(def.Default) {
						dataFound = true
						default_ = def.Default[i]
					}
					if dataFound {
						if default_ != nil {
							expects += formatArg(""+name+"", default_)
							rawExpects += "<" + name + "=" + repr(default_) + ">"
						} else {
							expects += formatArg(""+name+"", nil)
							rawExpects += "<" + name + ">"
						}
					} else {
						noDataCounter += 1
					}
				}
			}

			if len(filters) > 0 {
				match := false
				fullKeyMatch := strings.Join(def.Keys, " ")
				for _, f := range filters {
					if strings.Contains(fullKeyMatch, f) || strings.Contains(strings.ToLower(def.Description), strings.ToLower(f)) {
						match = true
						break
					}
				}
				if !match {
					continue
				}
			}

			kPadding := col1Width + (len(keysStr) - rawKeysLen)
			ePadding := col2Width + (len(expects) - len(rawExpects))

			fmt.Printf("  %-*s %-*s %s\n", kPadding, keysStr, ePadding, expects, def.Description)

			if def.AllowDupes || def.VarArgs || def.AfterCount > 1 || len(def.ExampleArgs) > 0 || len(def.Default) > 0 {
				fmt.Printf("    %sExample:\n    %s%s\n", Gray, formatExample(def), Reset)
			}
		}
	}
	println(Yellow + "Main" + Reset + " Usage Options:")
	parseThisData(ParseArgsData["MAIN"])
	for pkgName, defs := range ParseArgsData {
		if pkgName == "MAIN" {
			continue
		}
		println(Yellow + pkgName + Reset + " Usage Options:" + Reset)
		parseThisData(defs)
	}
}

func repr(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%#v", v)

}
func formatArg(name string, default_ any) string {
	if default_ != nil {
		return Yellow + "<" + Cyan + name + Yellow + "=" + Blue + repr(default_) + Yellow + ">" + Reset
	}
	return Yellow + "<" + Cyan + name + Yellow + ">" + Reset
}

func formatExample(def ArgumentData) string {
	p := "-"
	if len(def.Keys[0]) > 1 {
		p = "--"
	}
	rawKey := p + def.Keys[0]

	coloredKey := Blue + rawKey + Gray

	getValName := func(index int) string {
		name := fmt.Sprintf("val%d", index+1)
		var default_ any = nil
		if index < len(def.ExampleArgs) {
			name = def.ExampleArgs[index]
		}
		if index < len(def.Default) {
			default_ = def.Default[index]
		}
		if default_ != nil {
			return Blue + repr(default_) + Reset
		}
		return Yellow + "<" + Cyan + name + Yellow + ">" + Reset
	}

	if def.VarArgs {
		v1, v2, v3 := getValName(0), getValName(1), getValName(2)
		if def.AllowDupes {
			return fmt.Sprintf("%s %s %s %s\n    %s %s %s %s %s",
				coloredKey, v1, v2, v3,
				coloredKey, v1, v2, coloredKey, v3)
		}
		return fmt.Sprintf("%s %s %s %s", coloredKey, v1, v2, v3)
	}

	if def.AllowDupes && def.AfterCount == 0 {
		return fmt.Sprintf("%s %s %s (counts occurrences)",
			coloredKey, coloredKey, coloredKey)
	}

	if def.AllowDupes && def.AfterCount == 1 {
		v1, v2 := getValName(0), getValName(1)
		return fmt.Sprintf("%s %s %s %s", coloredKey, v1, coloredKey, v2)
	} else if def.AfterCount == 1 {
		v1 := getValName(0)
		return fmt.Sprintf("%s %s", coloredKey, v1)
	}

	if def.AfterCount > 1 {
		vals := []string{}
		for i := 0; i < def.AfterCount; i++ {
			vals = append(vals, getValName(i))
		}
		return fmt.Sprintf("%s %s", coloredKey, strings.Join(vals, " "))
	}
	return ""
}

func matches(keys []string, input string) bool {
	clean := strings.TrimLeft(input, "-")
	return slices.Contains(keys, clean)
}
