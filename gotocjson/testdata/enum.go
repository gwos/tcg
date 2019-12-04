// "enum" is just for testing our parsing, including in particular
// our handling of iota in enumeration values

package enum

// MetricKind: The metric kind of the time series.
//   "METRIC_KIND_UNSPECIFIED" - Do not use this default value.
//   "GAUGE" - An instantaneous measurement of a value.
//   "DELTA" - The change in a value during a time interval.
//   "CUMULATIVE" - A value accumulated over a time interval. Cumulative
type MetricKindEnum int

const (
	GAUGE                   MetricKindEnum = iota
	DELTA                                  
	CUMULATIVE                             
	METRIC_KIND_UNSPECIFIED                
)

type IntegerFlags int

const (
	first = 1 << iota
	second
	third
	fourth
)

type ValueTypeEnum string

const (
	IntegerType     ValueTypeEnum = "IntegerType"
	DoubleType                    = "DoubleType"
	StringType                    = "StringType"
	BooleanType                   = "BooleanType"
	TimeType                      = "TimeType"
	UnspecifiedType               = "UnspecifiedType"
)

type Inner struct {
    X int `json:"x"`
    Y int `json:"y"`
}

type Outer struct {
    Inner `json:"inner"`
    *Inner `json:"ptr_inner"`
}

// doc for the dummy function
// more doc, too
func standin(a int) int {
    return a
}
