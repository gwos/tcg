package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"github.com/spf13/pflag"
)

type Cfg struct {
	Folders  []string
	NoPrefix bool
	Verbose  int8
}

var (
	cfg     = new(Cfg)
	cfgOnce sync.Once

	logError, logWarn, logInfo, logDebug = func(flags int, calldepth int, level *int8) (
		func(format string, a ...any),
		func(format string, a ...any),
		func(format string, a ...any),
		func(format string, a ...any),
	) {
		loggerErr := log.New(log.Writer(), "ERR ", flags)
		loggerWrn := log.New(log.Writer(), "WRN ", flags)
		loggerInf := log.New(log.Writer(), "INF ", flags)
		loggerDbg := log.New(log.Writer(), "DBG ", flags)
		return func(format string, a ...any) {
				_ = loggerErr.Output(calldepth, fmt.Sprintf(format, a...))
			},
			func(format string, a ...any) {
				if *level > 0 {
					_ = loggerWrn.Output(calldepth, fmt.Sprintf(format, a...))
				}
			},
			func(format string, a ...any) {
				if *level > 1 {
					_ = loggerInf.Output(calldepth, fmt.Sprintf(format, a...))
				}
			},
			func(format string, a ...any) {
				if *level > 2 {
					_ = loggerDbg.Output(calldepth, fmt.Sprintf(format, a...))
				}
			}
	}(log.Lmsgprefix|log.Lshortfile|log.Ldate|log.Ltime|log.LUTC, 2, &(cfg.Verbose))
)

func getCfg() *Cfg {
	cfgOnce.Do(func() {
		flags := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
		flags.SortFlags = false
		folders := flags.StringSlice("folders", nil, "Folder pathes")
		noPrefix := flags.Bool("no-prefix", false, "Omit prefixing constant name")
		verbose := flags.Int8("verbose", 2, "Verbose level 0..3")
		if err := flags.Parse(os.Args[1:]); err != nil {
			logError("could not parse options: %s", err)
		}
		cfg.Folders = *folders
		cfg.NoPrefix = *noPrefix
		if *verbose < 0 {
			*verbose = 0
		}
		if *verbose > 3 {
			*verbose = 3
		}
		cfg.Verbose = *verbose
		logInfo("starting with options: %+v", *cfg)
	})
	return cfg
}

func isGoFile(fi fs.FileInfo) bool {
	return strings.HasSuffix(fi.Name(), ".go") &&
		!strings.HasSuffix(fi.Name(), "_test.go")
}

func writeHFile(name string, cc []*doc.Value) {
	name = formatConstName(name)
	fname := strings.ToLower(name) + ".h"
	f, err := os.Create(fname)
	if err != nil {
		logError("could not create file: %s: %s", fname, err)
		return
	}
	defer f.Close()
	logInfo("create file: %s", fname)

	fmt.Fprintf(f, "#ifndef %s_H\n#define %s_H\n", name, name)
	prefix := name + "_"
	if getCfg().NoPrefix {
		prefix = ""
	}
	for _, d := range cc {
		writeConst(f, d, prefix)
	}
	fmt.Fprintf(f, "\n#endif /* %s_H */\n", name)
}

func writeConst(w io.Writer, d *doc.Value, prefix string) {
	if d.Decl.Tok != token.CONST {
		return
	}
	if len(d.Doc) > 0 {
		fmt.Fprintf(w, "\n/* %s */\n", strings.Trim(d.Doc, "\n"))
	}
	for _, spec := range d.Decl.Specs {
		s := spec.(*ast.ValueSpec)
		if len(s.Names) != 1 {
			logWarn("len(spec.Names) != 1; spec: %+v", spec)
		}
		if s.Doc != nil {
			fmt.Fprintf(w, "\n/* %s */\n", strings.Trim(s.Doc.Text(), "\n"))
		}
		v := s.Values[0].(*ast.BasicLit)
		fmt.Fprintf(w, "#define %s%s %s\n",
			prefix,
			formatConstName(s.Names[0].String()),
			v.Value)
	}
}

func formatConstName(s string) string {
	if len(s) < 2 {
		return strings.ToUpper(s)
	}
	var b strings.Builder
	for i, c := range s {
		b.WriteRune(unicode.ToUpper(c))
		if i < len(s)-1 &&
			unicode.IsLetter(c) && unicode.IsLower(c) &&
			unicode.IsLetter(rune(s[i+1])) && unicode.IsUpper(rune(s[i+1])) {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func main() {
	_ = getCfg()

	/* inspired by headscan
	https://cs.opensource.google/go/go/+/refs/tags/go1.16:src/go/doc/headscan.go */
	for _, root := range cfg.Folders {
		fset := token.NewFileSet()
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				logError("could not parse dir: %s: %s", root, e)
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			pkgs, err := parser.ParseDir(fset, path, isGoFile, parser.ParseComments)
			if err != nil {
				logError("could not parse dir: %s: %s", root, err)
				return nil
			}
			for _, pkg := range pkgs {
				d := doc.New(pkg, path, doc.Mode(0))
				logDebug("package: %+v; doc: %+v", pkg, d)

				var cc []*doc.Value
				cc = append(cc, d.Consts...)
				for _, d := range d.Types {
					cc = append(cc, d.Consts...)
				}
				if len(cc) > 0 {
					writeHFile(pkg.Name, cc)
				}
			}
			return nil
		})
		if err != nil {
			logError("exiting due to error: %s", err)
			os.Exit(1)
		}
	}
}
