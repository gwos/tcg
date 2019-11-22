package customTime

import (
    "fmt"
    "time"
)

const (
    Layout = "2006-01-02T15:04:05.000-0700"
)

type CustomTime struct {
    time.Time
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *CustomTime) UnmarshalJSON(input []byte) error {
    strInput := string(input)

    customType, err := time.Parse(Layout, strInput)
    if err != nil {
        return err
    }

    *t = CustomTime{customType}
    return nil
}

// MarshalJSON implements json.Marshaler.
func (t CustomTime) MarshalJSON() ([]byte, error) {
    return []byte(fmt.Sprintf("%s", t.Format(Layout))), nil
}
