package types

import "time"

type HumanTime time.Time

func (ct HumanTime) MarshalJSON() ([]byte, error) {
    formatted := time.Time(ct).Format("2006-01-02 15:04")
    return []byte(`"` + formatted + `"`), nil
}