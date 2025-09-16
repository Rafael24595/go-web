package utils

import (
	"strconv"
	"strings"
)

type Argument struct {
	item string
}

func ArgumentFrom(item string) *Argument {
	return &Argument{
		item: item,
	}
}

func (a Argument) Bool() (bool, bool) {
	val, err := strconv.ParseBool(strings.ToLower(a.item))
	if err != nil {
		return false, false
	}
	return val, true
}

func (a Argument) Boold(def bool) bool {
	if res, ok := a.Bool(); ok {
		return res
	}
	return def
}

func (a Argument) String() string {
	return a.item
}

func (a Argument) Int() (int, bool) {
	val, err := strconv.Atoi(a.item)
	if err != nil {
		return 0, false
	}
	return val, true
}

func (a Argument) Intd(def int) int {
	if res, ok := a.Int(); ok {
		return res
	}
	return def
}

func (a Argument) Int64() (int64, bool) {
	val, err := strconv.ParseInt(a.item, 10, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func (a Argument) Int64d(def int64) int64 {
	if res, ok := a.Int64(); ok {
		return res
	}
	return def
}

func (a Argument) Float32() (float32, bool) {
	val, err := strconv.ParseFloat(a.item, 32)
	if err != nil {
		return 0, false
	}
	return float32(val), true
}

func (a Argument) Float32d(def float32) float32 {
	if res, ok := a.Float32(); ok {
		return res
	}
	return def
}

func (a Argument) Float64() (float64, bool) {
	val, err := strconv.ParseFloat(a.item, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}

func (a Argument) Float64d(def float64) float64 {
	if res, ok := a.Float64(); ok {
		return res
	}
	return def
}
