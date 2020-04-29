package connectors

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
func MB(bytes float64) float64 {
	return bytes / 1048576
}

//
// Converts byte value to Kilobyte value
//
// @param bytes - value in bytes
// @return value in Kilobytes
func KB(bytes float64) float64 {
	return bytes / 1024
}

//
// Converts byte value to Gigabyte value
//
// @param bytes - value in bytes
// @return value in Gigabytes
func GB(bytes float64) float64 {
	return bytes / 1073741824
}

//
// Converts byte value to Terabyte value
//
// @param bytes - value in bytes
// @return value in Terabytes
func TB(bytes float64) float64 {
	return bytes / 1099511627776
}

func MaxInt(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func MinInt(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxFloat(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func MinFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func IntToDouble(value int) float64 {
	return float64(value)
}

func DoubleToInt(value float64) int64 {
	return int64(value)
}

func MB2(bytes float64) float64 {
	return bytes / 1000000
}

func KB2(bytes float64) float64 {
	return bytes / 1000
}

func GB2(bytes float64) float64 {
	return bytes / 1000000000
}

func TB2(bytes float64) float64 {
	return bytes / 1000000000000
}

//
// Turns a number such as .87 into an integer percentage (87). Also handles rounding of percentages
//
// @param value - the value to be rounded to a full integer percentage
// @return the percentage value as an integer
func ToPercentage(value float64) int64 {
	result := value * 100
	result = MaxFloat(0.0, result)
	return int64(result + 0.49)
}

func ToPercentageLimit(value float64) int64 {
	result := value * 100
	result = MaxFloat(0.0, MinFloat(100.0, result))
	return int64(result + 0.49)
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
func DivideToPercentage(dividend float64, divisor float64) int64 {
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
func PercentageUsed(used, available float64) int64 {
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
func PercentageUnused(unused float64, available float64) int64 {
	demand := unused
	usage := available
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
func ScalePercentageUnused(unused, available, usageScaleFactor, availableScaleFactor float64) int64 {
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
func ScalePercentageUsed(used, available, usedScaleFactor, availableScaleFactor float64) int64 {
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
