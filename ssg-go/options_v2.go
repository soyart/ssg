package ssg

import (
	"fmt"
	"os"
)

type OptionType string

const (
	OptionTypeConfig       = "ssg-config"
	OptionTypeHook         = "ssg-hook"
	OptionTypeHookGenerate = "ssg-hook-generate"
	OptionTypePipeline     = "ssg-pipeline"
)

type OptionV2 interface {
	Option() Option
	OptionName() string
	OptionType() OptionType
}

func (s *Ssg) WithV2(opts ...OptionV2) *Ssg {
	types := make(map[OptionType]string)
	for _, o := range opts {
		t := o.OptionType()
		if t == "" {
			panic("found empty option type")
		}

		Fprintf(os.Stdout, "applying option of type '%s': '%s'", t, o.OptionName())

		prev, ok := types[t]
		if ok {
			Fprintf(os.Stdout, "found duplicate option type '%s' (already applied option '%s', but also found option '%s')", t, prev, o.OptionName())
			panic("found duplicate option type")
		}

		o.Option()(s)
		s.options.options = append(s.options.options, o)
		Fprintf(os.Stdout, "applied option of type '%s': '%s'", t, o.OptionName())
	}
	return s
}

func CachingV2() OptionV2 {
	return optionCaching{}
}
func WritersFromEnvV2() OptionV2 {
	return optionWritersFromEnv{}
}
func WithHookV2(hook Hook, description string) OptionV2 {
	return optionHook{h: hook, s: description}
}

type optionCaching struct{}

func (o optionCaching) Option() Option         { return Caching() }
func (o optionCaching) OptionName() string     { return "option-caching" }
func (o optionCaching) OptionType() OptionType { return OptionTypeConfig }

type optionWritersFromEnv struct{}

func (o optionWritersFromEnv) Option() Option         { return WritersFromEnv() }
func (o optionWritersFromEnv) OptionName() string     { return "option-writers-from-env" }
func (o optionWritersFromEnv) OptionType() OptionType { return OptionTypeConfig }

type optionHook struct {
	h Hook
	s string
}

func (o optionHook) Option() Option { return WithHook(o.h) }
func (o optionHook) OptionName() string {
	if o.s != "" {
		return fmt.Sprintf("option-hook-%s", o.s)
	}

	return "option-hook-unknown"
}

func (o optionHook) OptionType() OptionType { return OptionTypeHook }

type optionHookGenerate struct {
	h HookGenerate
	s string
}

func (o optionHookGenerate) Option() Option { return WithHookGenerate(o.h) }
func (o optionHookGenerate) OptionName() string {
	if o.s != "" {
		return fmt.Sprintf("option-hook-generate-%s", o.s)
	}
	return "option-hook-generate-unknown"
}

func (o optionHookGenerate) OptionType() OptionType { return OptionTypeHookGenerate }

type optionPipelines struct {
	pipes []Pipeline
}

func (o optionPipelines) Option() Option {
	return WithPipelines(o.pipes)
}
func (o optionPipelines) OptionName() string {
	return ""
}
func (o optionPipelines) OptionType() OptionType {
	return OptionTypePipeline
}
