package ssg

type Outputs interface {
	AddOutputs(...OutputFile)
}

type outputsV1 struct {
	stream chan<- OutputFile
}

func NewOutputs(c chan<- OutputFile) Outputs {
	return outputsV1{stream: c}
}

func (o outputsV1) AddOutputs(outputs ...OutputFile) {
	for i := range outputs {
		o.stream <- outputs[i]
	}
}
