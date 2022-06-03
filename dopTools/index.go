package dopTools

import (
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"strings"
	"syscall"

	"github.com/rendau/dop/dopErrs"
	"github.com/rendau/dop/dopTypes"
	"github.com/spf13/viper"
)

var (
	phoneRegexp = regexp.MustCompile(`^[1-9][0-9]{10,30}$`)
	emailRegexp = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,10}$`)

	defaultMaxPageSize int64 = 100
)

func RequirePageSize(pars dopTypes.ListParams, maxPageSize int64) error {
	if maxPageSize == 0 {
		maxPageSize = defaultMaxPageSize
	}

	if pars.PageSize == 0 || pars.PageSize > maxPageSize {
		return dopErrs.IncorrectPageSize
	}

	return nil
}

func NormalizePhone(p string) string {
	l := len(p)
	if l > 1 {
		if p[0] == '+' {
			p = p[1:]
		} else {
			if l == 10 && p[0] == '7' {
				p = "7" + p
			} else if l == 11 && strings.HasPrefix(p, "87") {
				p = "7" + p[1:]
			}
		}
	}
	return p
}

func ValidatePhone(v string) bool {
	return phoneRegexp.MatchString(v)
}

func ValidateEmail(v string) bool {
	return emailRegexp.MatchString(v)
}

func Coalesce[T any](v *T, nv T) T {
	if v == nil {
		return nv
	}

	return *v
}

func NewPtr[T any](v T) *T {
	return &v
}

func NewSlicePtr[T any](v ...T) *[]T {
	res := make([]T, 0, len(v))
	res = append(res, v...)
	return &res
}

func SliceHasValue[T comparable](sl []T, v T) bool {
	for _, x := range sl {
		if x == v {
			return true
		}
	}

	return false
}

func SlicesAreSame[T comparable](a, b []T) bool {
	for _, x := range a {
		if !SliceHasValue(b, x) {
			return false
		}
	}

	for _, x := range b {
		if !SliceHasValue(a, x) {
			return false
		}
	}

	return true
}

func SlicesIntersection[T comparable](sl1, sl2 []T) []T {
	result := make([]T, 0)

	if len(sl1) == 0 || len(sl2) == 0 {
		return result
	}

	for _, x := range sl1 {
		if SliceHasValue(sl2, x) {
			result = append(result, x)
		}
	}

	return result
}

func SliceExcludeValues[T comparable](sl, vs []T) []T {
	result := make([]T, 0, len(sl))

	for _, x := range sl {
		if !SliceHasValue(vs, x) {
			result = append(result, x)
		}
	}

	return result
}

func StopSignal() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	return ch
}

func SetViperDefaultsFromObj(obj any) {
	v := reflect.Indirect(reflect.ValueOf(obj))
	fields := reflect.VisibleFields(v.Type())

	var fieldTag string
	var tagName string

	for _, field := range fields {
		if field.Anonymous || !field.IsExported() {
			continue
		}

		fieldTag = field.Tag.Get("mapstructure")
		if fieldTag == "" {
			continue
		}

		tagName = strings.SplitN(fieldTag, ",", 2)[0]

		viper.SetDefault(tagName, "")
	}
}
