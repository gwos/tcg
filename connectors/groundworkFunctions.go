package connectors

import (
	"errors"
	"fmt"
	"github.com/PaesslerAG/gval"
	"strings"
)

const (
	Kb                 = "GW:KB"
	Mb                 = "GW:MB"
	Gb                 = "GW:GB"
	Tb                 = "GW:TB"
	Kb2                = "GW:KB2"
	Mb2                = "GW:MB2"
	Gb2                = "GW:GB2"
	Tb2                = "GW:TB2"
	IntMax             = "GW:maxInt"
	IntMin             = "GW:minInt"
	DoubleMax          = "GW:maxDouble"
	DoubleMin          = "GW:minDouble"
	IntDouble          = "GW:toDouble"
	DoubleInt          = "GW:toInt"
	ToPercent          = "GW:toPercentage"
	PercentUsed        = "GW:percentageUsed"
	PercentUnused      = "GW:percentageUnused"
	ToPercentLimit     = "GW:toPercentageLimit"
	DivideToPercent    = "GW:divideToPercentage"
	ScalePercentUsed   = "GW:scalePercentageUsed"
	ScalePercentUnused = "GW:scalePercentageUnused"
)

var expressionToFuncMap = map[string]interface{}{
	Kb:                 KB,
	Mb:                 MB,
	Gb:                 GB,
	Tb:                 TB,
	Kb2:                KB2,
	Mb2:                MB2,
	Gb2:                GB2,
	Tb2:                TB2,
	IntMax:             MaxInt,
	IntMin:             MinInt,
	DoubleMax:          MaxFloat,
	DoubleMin:          MinFloat,
	IntDouble:          IntToDouble,
	DoubleInt:          DoubleToInt,
	ToPercent:          ToPercentage,
	PercentUsed:        PercentageUsed,
	PercentUnused:      PercentageUnused,
	ToPercentLimit:     ToPercentageLimit,
	DivideToPercent:    DivideToPercentage,
	ScalePercentUsed:   ScalePercentageUsed,
	ScalePercentUnused: ScalePercentageUnused,
}

var expressionToArgsCountMap = map[string]int{
	Kb:                 1,
	Mb:                 1,
	Gb:                 1,
	Tb:                 1,
	Kb2:                1,
	Mb2:                1,
	Gb2:                1,
	Tb2:                1,
	IntMax:             2,
	IntMin:             2,
	DoubleMax:          2,
	DoubleMin:          2,
	IntDouble:          2,
	DoubleInt:          2,
	ToPercent:          1,
	PercentUsed:        2,
	PercentUnused:      2,
	ToPercentLimit:     1,
	DivideToPercent:    2,
	ScalePercentUsed:   4,
	ScalePercentUnused: 4,
}

/*
   1 Kilobyte = 1,024 Bytes
   1 Megabyte = 1,048,576 Bytes
   1 Gigabyte = 1,073,741,824 Bytes
   1 Terabyte = 1,099,511,627,776 Bytes
*/

//
// Converts byte value to Megabyte value
//
// @param bytes - value in bytes
// @return value in Megabytes
func MB(values ...float64) float64 {
	bytes := values[0]
	return bytes / 1048576
}

//
// Converts byte value to Kilobyte value
//
// @param bytes - value in bytes
// @return value in Kilobytes
func KB(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1024
}

//
// Converts byte value to Gigabyte value
//
// @param bytes - value in bytes
// @return value in Gigabytes
func GB(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1073741824
}

//
// Converts byte value to Terabyte value
//
// @param bytes - value in bytes
// @return value in Terabytes
func TB(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1099511627776
}

func MaxInt(values ...float64) float64 {
	a := values[0]
	b := values[1]

	if a > b {
		return a
	}
	return b
}

func MinInt(values ...float64) float64 {
	a := values[0]
	b := values[1]

	if a < b {
		return a
	}
	return b
}

func MaxFloat(values ...float64) float64 {
	a := values[0]
	b := values[1]

	if a > b {
		return a
	}
	return b
}

func MinFloat(values ...float64) float64 {
	a := values[0]
	b := values[1]

	if a < b {
		return a
	}
	return b
}

func IntToDouble(values ...float64) float64 {
	return values[0]
}

func DoubleToInt(values ...float64) float64 {
	return values[0]
}

func MB2(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1000000
}

func KB2(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1000
}

func GB2(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1000000000
}

func TB2(values ...float64) float64 {
	bytes := values[0]

	return bytes / 1000000000000
}

//
// Turns a number such as .87 into an integer percentage (87). Also handles rounding of percentages
//
// @param value - the value to be rounded to a full integer percentage
// @return the percentage value as an integer
func ToPercentage(values ...float64) float64 {
	value := values[0]

	result := value * 100
	result = MaxFloat(0.0, result)
	return result + 0.49
}

func ToPercentageLimit(values ...float64) float64 {
	value := values[0]

	result := value * 100
	result = MaxFloat(0.0, MinFloat(100.0, result))
	return result + 0.49
}

//
// Given two metrics, <code>dividend</code> and <code>divisor</code> divides them and returns a percentage ratio
//
// Example:
//
// GW:divideToPercentage(summary.quickStats.overallMemoryUsage,summary.hardware.memorySize)
//
// @param - dividend typically a usage or free type metric
// @param - divisor typically a totality type metric, such as total disk space
// @return The percentage ratio as an integer
func DivideToPercentage(values ...float64) float64 {
	dividend := values[0]
	divisor := values[1]

	if divisor == 0 {
		return 0
	}
	return ToPercentage(dividend / divisor)
}

//
// This Function provides percentage usage synthetic values.
// Calculates the usage percentage for a given <code>used</code> metric and a corresponding <code>available</code> metric.
//
// Example:
//
// 		PercentageUsed(summary.quickStats.overallMemoryUsage, summary.hardware.memorySize)
//
// @param - used Represents a 'used' metric value of how much of this resource has been used such as 'overallMemoryUsage'
// @param - available Represents the totality of a resource, such as all memory available
// @return The percentage usage as an integer
func PercentageUsed(values ...float64) float64 {
	used := values[0]
	available := values[1]

	return ScalePercentageUsed(used, available, 1.0, 0)
}

//
// This Function provides percentage unused/free synthetic values.
// Calculates the unused(free) percentage for a given <code>unused</code> metric and a corresponding <code>available</code> metric.
// Both the unused metric and available metric can be scaled by corresponding scale factor parameters.
//
// Example:
//
// 		PercentageUnused(summary.freeSpace, summary.capacity)
//
// @param unused - Represents a metric reference value of how much of this resource has not be used (free)
// @param available - Represents the totality of a resource, such as all disk space available
// @return The percentage not used (free) as an integer
func PercentageUnused(values ...float64) float64 {
	demand := values[0]
	usage := values[1]
	return ScalePercentageUnused(demand, usage, 1.0, 0)
}

//
// This Function provides percentage unused/free synthetic values.
// Calculates the unused(free) percentage for a given <code>unused</code> metric and a corresponding <code>available</code> metric.
// Both the unused metric and available metric can be scaled by corresponding scale factor parameters.
//
// Example:
//
// 		scalePercentageUnused(summary.freeSpace,summary.capacity, 1.0, null, true)
//
// @param unused  Represents a metric reference value of how much of this resource has not be used (free)
// @param available Represents the totality of a resource, such as all disk space available
// @param usageScaleFactor multiply usage parameter by this value, or pass in null to not scale. Passing in 1.0 will also not scale
// @param availableScaleFactor multiply available parameter by this value, or pass in null to not scale. Passing in 1.0 will also not scale
// @return The percentage not used (free) as an integer
func ScalePercentageUnused(values ...float64) float64 {
	unused := values[0]
	available := values[1]
	usageScaleFactor := values[2]
	availableScaleFactor := values[3]

	if unused == 0 && available == 0 {
		return 0
	}

	var usage float64
	if usageScaleFactor == 0 {
		usage = unused
	} else {
		usage = unused * usageScaleFactor
	}

	var availableScaled float64
	if availableScaleFactor == 0 {
		availableScaled = available
	} else {
		availableScaled = available * availableScaleFactor
	}

	if usage != 0 {
		usage = usage / availableScaled
	}

	usage = 1.0 - usage

	return ToPercentage(usage)
}

//
// This Function provides percentage usage synthetic values.
// Calculates the usage percentage for a given <code>used</code> metric and a corresponding <code>available</code> metric.
// Both the used metric and available metric can be scaled by corresponding scale factor parameters.
//
// Example:
//
// 		scalePercentageUsed(summary.quickStats.overallMemoryUsage,summary.hardware.memorySize, 1.0, 1.0)
//
// @param used Represents a 'used' metric value of how much of this resource has been used such as 'overallMemoryUsage'
// @param available Represents the totality of a resource, such as all memory available
// @param usedScaleFactor multiply usage parameter by this value, or pass in null to not scale. Passing in 1.0 will also not scale
// @param availableScaleFactor multiply available parameter by this value, or pass in null to not scale. Passing in 1.0 will also not scale
// @return The percentage usage as an integer
func ScalePercentageUsed(values ...float64) float64 {
	used := values[0]
	available := values[1]
	usedScaleFactor := values[2]
	availableScaleFactor := values[3]

	if used == 0 && available == 0 {
		return 0
	}

	var usage float64
	if usedScaleFactor == 0 {
		usage = used
	} else {
		usage = used * usedScaleFactor
	}

	var availableScaled float64
	if availableScaleFactor == 0 {
		availableScaled = available
	} else {
		availableScaled = available * availableScaleFactor
	}

	if usage != 0 {
		usage = usage / availableScaled
	}

	return ToPercentage(usage)
}

func EvaluateGroundworkFunction(expression string, vars map[string]interface{}) (float64, []float64, error) {
	expression = strings.TrimSpace(expression)
	if strings.HasPrefix(expression, "GW") && strings.HasSuffix(expression, ")") {
		for {
			gwFuncName := expression[:strings.Index(expression, "(")]
			exp := expression[strings.Index(expression, "(")+1 : strings.LastIndex(expression, ")")]

			if function, exists := expressionToFuncMap[gwFuncName]; exists {
				if _, values, err := EvaluateGroundworkFunction(exp, vars); err == nil {
					if len(values) != expressionToArgsCountMap[gwFuncName] {
						return -1, nil, errors.New(fmt.Sprintf("Too much arguments for Groundwork function [%s]", gwFuncName))
					}
					v := function.(func(...float64) float64)(values...)
					return v, []float64{v}, nil
				} else {
					return -1, nil, err
				}
			} else {
				return -1, nil, errors.New(fmt.Sprintf("Groundwork function [%s] doesn't exist", gwFuncName))
			}
		}
	} else {
		funcArgs := strings.Split(expression, ",")
		var result []float64
		for _, val := range funcArgs {
			if strings.HasPrefix(val, "GW") {
				if v, _, err := EvaluateGroundworkFunction(val, vars); err == nil {
					result = append(result, v)
					continue
				} else {
					return -1, nil, err
				}
			}
			if strings.ContainsAny(val, "+-/*") {
				if v, err := gval.Evaluate(strings.ReplaceAll(val, ".", "_"), vars); err == nil {
					result = append(result, v.(float64))
					continue
				} else {
					return -1, nil, err
				}
			}
			if v, ok := vars[strings.ReplaceAll(strings.TrimSpace(val), ".", "_")]; ok {
				result = append(result, v.(float64))
				continue
			} else {
				return -1, nil, errors.New(fmt.Sprintf("Undefined variable %s", val))
			}
		}
		return -1, result, nil
	}
}

func ListExpressions(name string) []Expression {
	expressions := []Expression{}

	for key, argsCount := range expressionToArgsCountMap {
		if strings.Contains(key, name) {
			expressions = append(expressions, Expression{
				key, argsCount,
			})
		}
	}

	return expressions
}

type Expression struct {
	Name      string `json:"name"`
	ArgsCount int    `json:"argsCount"`
}
