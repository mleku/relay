// Package unix is a json.Marshal/json.Unmarshal implementation for timestamps that are stored
// in the JSON as 1 second precision integers.
package unix

import (
	"time"

	"relay.mleku.dev/ints"
)

type Timestamp struct{ time.Time }

func UnixNow() *Timestamp { return &Timestamp{Time: time.Now()} }

func (u *Timestamp) MarshalJSON() (b []byte, err error) {
	b = ints.New(u.Time.Unix()).Marshal(b)
	return
}

func (u *Timestamp) UnmarshalJSON(b []byte) (err error) {
	t := ints.New(0)
	_, err = t.Unmarshal(b)
	u.Time = time.Unix(int64(t.N), 0)
	return
}

type TimestampMilli struct{ time.Time }

func (u *TimestampMilli) MarshalJSON() (b []byte, err error) {
	b = ints.New(u.Time.UnixMilli()).Marshal(b)
	return
}

func (u *TimestampMilli) UnmarshalJSON(b []byte) (err error) {
	t := ints.New(0)
	_, err = t.Unmarshal(b)
	u.Time = time.UnixMilli(int64(t.N))
	return
}

type Time struct{ time.Time }

func Now() *Time { return &Time{Time: time.Now()} }

func (u *Time) MarshalJSON() (b []byte, err error) {
	b = ints.New(u.Time.Unix()).Marshal(b)
	return
}

func (u *Time) UnmarshalJSON(b []byte) (err error) {
	t := ints.New(0)
	_, err = t.Unmarshal(b)
	u.Time = time.Unix(int64(t.N), 0)
	return
}
