package ssg

import "os"

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
		prev, ok := types[t]
		if ok {
			Fprintf(os.Stdout, "found duplicate option type '%s' (already applied option '%s', but also found option '%s')", t, prev, o.OptionName())
		}

		Fprintf(os.Stdout, "applying option of type '%s': '%s'", t, o.OptionName())
		o.Option()(s)
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

type optionCaching struct{}

func (o optionCaching) Option() Option         { return Caching() }
func (o optionCaching) OptionName() string     { return "option-caching" }
func (o optionCaching) OptionType() OptionType { return OptionTypeConfig }

type optionWritersFromEnv struct{}

func (o optionWritersFromEnv) Option() Option         { return WritersFromEnv() }
func (o optionWritersFromEnv) OptionName() string     { return "option-writers-from-env" }
func (o optionWritersFromEnv) OptionType() OptionType { return OptionTypeConfig }
