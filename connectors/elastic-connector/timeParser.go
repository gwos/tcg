package main

import (
	"github.com/gwos/tng/log"
	"strconv"
	"strings"
	"time"
)

const layout = "2006-01-02T15:04:05.000Z"
const now = "now"

func parseTime(timeString string, isStartTime bool, location *time.Location) time.Time {
	if strings.Contains(timeString, now) {
		return parseTimeExpression(timeString, isStartTime, location)
	}
	result, err := time.Parse(layout, timeString)
	if err != nil {
		log.Error(err)
	}
	return result.In(location)
}

// Converts relative expressions such as "now-5d" to Time
func parseTimeExpression(expression string, isStartTime bool, location *time.Location) time.Time {
	timeNow := time.Now().In(location)
	if expression == now {
		return timeNow
	}
	// character after "now" is operator (+/-)
	operator := expression[len(now) : len(now)+1]
	// everything after "now" and operator is a relative part
	relativePart := expression[len(now)+1:]
	var rounded = false
	// rounded expression ends with "/y", "/M", "/w", "/d", "/h", "/m", "/s"
	if strings.Contains(relativePart, "/") {
		// remove these last 2 characters from relative part, keep in mind that expression is rounded
		relativePart = relativePart[:len(relativePart)-2]
		rounded = true
	}
	// the last character of the relative part defines period ("y", "M", "w", "d", "h", "m", "s"), everything else is interval
	interval := relativePart[:len(relativePart)-1]
	period := relativePart[len(relativePart)-1:]
	i, err := strconv.Atoi(interval)
	if operator == "-" {
		i = -i
	}
	if err != nil {
		log.Error("Error parsing time filter expression: %s", err)
	}
	var result time.Time
	switch period {
	case "y":
		result = timeNow.AddDate(i, 0, 0)
		if rounded {
			if isStartTime {
				// StartTime is being rounded to the beginning of period
				result = time.Date(result.Year(), 1, 1, 0, 0, 0, 0, location)
			} else {
				// EndTime is being rounded to the last millisecond of period
				// to achieve this we subtract one millisecond from the next period
				result = time.Date(result.Year()+1, 1, 1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "M":
		result = timeNow.AddDate(0, i, 0)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), 1, 0, 0, 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month()+1, 1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "w":
		dayOfDesiredWeek := timeNow.AddDate(0, 0, 7*i)
		if rounded {
			// week is being rounded to the beginning of past Sunday for StartTime filter and to the end of next Saturday for EndTime filter
			var offsetFromSunday int
			var offsetToSaturday int
			switch dayOfDesiredWeek.Weekday() {
			case time.Monday:
				offsetFromSunday = 1
				offsetToSaturday = 5
				break
			case time.Tuesday:
				offsetFromSunday = 2
				offsetToSaturday = 4
				break
			case time.Wednesday:
				offsetFromSunday = 3
				offsetToSaturday = 3
				break
			case time.Thursday:
				offsetFromSunday = 4
				offsetToSaturday = 2
				break
			case time.Friday:
				offsetFromSunday = 5
				offsetToSaturday = 1
				break
			case time.Saturday:
				offsetFromSunday = 6
				offsetToSaturday = 0
				break
			case time.Sunday:
				offsetFromSunday = 0
				offsetToSaturday = 6
				break
			}
			if isStartTime {
				result = time.Date(dayOfDesiredWeek.Year(), dayOfDesiredWeek.Month(), dayOfDesiredWeek.Day()-offsetFromSunday, 0, 0, 0, 0, location)
			} else {
				result = time.Date(dayOfDesiredWeek.Year(), dayOfDesiredWeek.Month(), dayOfDesiredWeek.Day()+offsetToSaturday+1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		} else {
			result = dayOfDesiredWeek
		}
		break
	case "d":
		result = timeNow.AddDate(0, 0, i)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), 0, 0, 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day()+1, 0, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
	case "h":
		result = timeNow.Add(time.Duration(i) * time.Hour)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), 0, 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour()+1, 0, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "m":
		result = timeNow.Add(time.Duration(i) * time.Minute)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), 0, 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute()+1, 0, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	case "s":
		result = timeNow.Add(time.Duration(i) * time.Second)
		if rounded {
			if isStartTime {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second(), 0, location)
			} else {
				result = time.Date(result.Year(), result.Month(), result.Day(), result.Hour(), result.Minute(), result.Second()+1, 0, location)
				result = result.Add(-1 * time.Millisecond)
			}
		}
		break
	default:
		log.Error("Error parsing time filter expression: unknown period format '" + period + "'")
	}
	return result
}
