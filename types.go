package main

import (
	"strings"
	"time"
)

type Date struct {
	time.Time
}

func (d Date) MarshalCSV() (string, error) {
	return d.Time.Format(time.DateOnly), nil
}

type Storage []string

func (s Storage) MarshalCSV() (string, error) {
	return strings.Join([]string(s), "\n"), nil
}
