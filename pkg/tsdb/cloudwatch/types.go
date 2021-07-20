package cloudwatch

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type cloudwatchResponse struct {
	DataFrames              data.Frames
	Id                      string
	RefId                   string
	Expression              string
	RequestExceededMaxLimit bool
	PartialData             bool
	Period                  int
}

type queryError struct {
	err   error
	RefID string
}

func (e *queryError) Error() string {
	return fmt.Sprintf("error parsing query %q, %s", e.RefID, e.err)
}

type cloudWatchLink struct {
	View    string        `json:"view"`
	Stacked bool          `json:"stacked"`
	Title   string        `json:"title"`
	Start   string        `json:"start"`
	End     string        `json:"end"`
	Region  string        `json:"region"`
	Metrics []interface{} `json:"metrics"`
}

type metricExpression struct {
	Expression string `json:"expression"`
}

type metricStatMeta struct {
	Stat   string `json:"stat"`
	Period int    `json:"period"`
}
