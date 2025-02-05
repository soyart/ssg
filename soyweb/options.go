package soyweb

import "github.com/soyart/ssg/ssg-go"

type OptionIndexGenerator struct{}

func (o OptionIndexGenerator) Option() ssg.Option {
	return ssg.WithPipelines(IndexGenerator)
}

func (o OptionIndexGenerator) OptionName() string         { return "option-soyweb-index-generator" }
func (o OptionIndexGenerator) OptionType() ssg.OptionType { return OptionTypeConfig }
