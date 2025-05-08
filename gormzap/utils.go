package gormzap

import (
	"runtime"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	gormPackage    = "gorm.io/gorm"
	gormzapPackage = "github.com/thinkgos/gin-contrib/gormzap"
)

type CallerCore struct {
	level        zap.AtomicLevel
	skip         int
	skipPackages []string
	caller       func(depth int, skipPackages ...string) zap.Field
}

func NewCallerCore() *CallerCore {
	return &CallerCore{
		level:        zap.NewAtomicLevelAt(zap.InfoLevel),
		skip:         2,
		skipPackages: nil,
		caller:       callerFile,
	}
}

// AddSkip add the number of callers skipped by caller annotation.
func (c *CallerCore) AddSkip(callerSkip int) *CallerCore {
	c.skip += callerSkip
	return c
}

// AddSkipPackage add the caller skip package.
func (c *CallerCore) AddSkipPackage(vs ...string) *CallerCore {
	c.skipPackages = append(c.skipPackages, vs...)
	return c
}

// SetLevel set the caller level.
func (c *CallerCore) SetLevel(lv zapcore.Level) *CallerCore {
	c.level.SetLevel(lv)
	return c
}

// Level returns the minimum enabled log level.
func (c *CallerCore) Level() zapcore.Level {
	return c.level.Level()
}

// Enabled returns true if the given level is at or above this level.
func (c *CallerCore) Enabled(lvl zapcore.Level) bool {
	return c.level.Enabled(lvl)
}

// UseExternalLevel use external level, which controller by user.
func (c *CallerCore) UseExternalLevel(l zap.AtomicLevel) *CallerCore {
	c.level = l
	return c
}

// callerFile caller file.
func callerFile(depth int, skipPackages ...string) zap.Field {
	var file string
	var line int
	var ok bool

	for i := depth; i < depth+15; i++ {
		_, file, line, ok = runtime.Caller(i)
		if ok && !skipPackage(file, skipPackages...) {
			break
		}
	}
	return zap.String("file", file+":"+strconv.FormatInt(int64(line), 10))
}

func skipPackage(file string, skipPackages ...string) bool {
	if strings.HasSuffix(file, "_test.go") {
		return false
	}
	if strings.Contains(file, gormPackage) ||
		strings.Contains(file, gormzapPackage) {
		return true
	}
	for _, p := range skipPackages {
		if strings.Contains(file, p) {
			return true
		}
	}
	return false
}
