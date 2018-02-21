package logger

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
)

// Logger default logger
type Logger struct {
	LogWriter *logrus.Entry
}

func NewLogger(log *logrus.Entry) *Logger {
	l := &Logger{
		LogWriter: log,
	}

	return l
}

// Print format & print log
func (l Logger) Print(values ...interface{}) {
	if values[0] == "sql" {
		toLog := SQLFormatter(values...)
		msg := toLog["sql"].(string)
		delete(toLog, "sql")
		l.LogWriter.WithFields(toLog).Debugln(msg)
	} else {
		l.LogWriter.Debugln(values[2:]...)
	}
}

var SQLFormatter = func(values ...interface{}) (messages map[string]interface{}) {
	if len(values) > 1 {
		var (
			sql             string
			duration        string
			formattedValues []string
			// level           = values[0]
			// currentTime     = time.Now().Format("2006-01-02 15:04:05")
			source = fmt.Sprintf("%v", values[1])
		)

		messages = map[string]interface{}{}

		// duration
		duration = fmt.Sprintf("%.2f", float64(values[2].(time.Duration).Nanoseconds()/1e4)/100.0)
		// sql

		for _, value := range values[4].([]interface{}) {
			indirectValue := reflect.Indirect(reflect.ValueOf(value))
			if indirectValue.IsValid() {
				value = indirectValue.Interface()
				if t, ok := value.(time.Time); ok {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
				} else if b, ok := value.([]byte); ok {
					if str := string(b); isPrintable(str) {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
					} else {
						formattedValues = append(formattedValues, "'<binary>'")
					}
				} else if r, ok := value.(driver.Valuer); ok {
					if value, err := r.Value(); err == nil && value != nil {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					} else {
						formattedValues = append(formattedValues, "NULL")
					}
				} else {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				}
			} else {
				formattedValues = append(formattedValues, "NULL")
			}
		}

		// differentiate between $n placeholders or else treat like ?
		if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
			sql = values[3].(string)
			for index, value := range formattedValues {
				placeholder := fmt.Sprintf(`\$%d`, index+1)
				sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value)
			}
		} else {
			formattedValuesLength := len(formattedValues)
			for index, value := range sqlRegexp.Split(values[3].(string), -1) {
				sql += value
				if index < formattedValuesLength {
					sql += formattedValues[index]
				}
			}
		}

		messages["duration"] = duration
		messages["sql"] = sql
		messages["source"] = source
	}

	return
}

var (
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
)

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
