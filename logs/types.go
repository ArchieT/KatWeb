package logs

import (
	"net/http"
	"fmt"
	"math"
)

type Level uint8

const (
	Warn Level = iota
	Info
)

func (l Level) String() string {
	switch l {
	case Warn:
		return "Warn"
	case Info:
		return "Info"
	default:
		return "XXXX"
	}
}

type EntryInterface interface {
	Level() Level
	Content() string
}

type Entry struct {
	EntryInterface
}

func (e Entry) String() string {
	return fmt.Sprint("[", e.Level(), "]", ": ", e.Content())
}

type FailedToContactGithubAPIforUpdate struct {
	Resp *http.Response
	Err  error
}

func (*FailedToContactGithubAPIforUpdate) Level() Level {
	return Warn
}

func (*FailedToContactGithubAPIforUpdate) Content() string {
	return "Failed to contact GitHub API for update!"
}

type FailedToReadGithubAPIUpdatesResponseBody struct {
	Err error
}

func (*FailedToReadGithubAPIUpdatesResponseBody) Level() Level {
	return Warn
}

func (*FailedToReadGithubAPIUpdatesResponseBody) Content() string {
	return "Failed to read Github API about update response body!"
}

type FailedToParseGithubAPIUpdatesResponse struct {
	Err error
}

func (*FailedToParseGithubAPIUpdatesResponse) Level() Level {
	return Warn
}

func (*FailedToParseGithubAPIUpdatesResponse) Content() string {
	return "Failed to parse Github API about update response body!"
}

type EmptyGithubAPIUpdatesResponse struct{}

func (*EmptyGithubAPIUpdatesResponse) Level() Level {
	return Warn
}

func (*EmptyGithubAPIUpdatesResponse) Content() string {
	return "Github API response about updates is empty!"
}

type CurrentOrUpdates bool

const (
	Latest  CurrentOrUpdates = true
	Current CurrentOrUpdates = false
)

func (c CurrentOrUpdates) String() string {
	switch c {
	case Current:
		return "current"
	case Latest:
		return "latest"
	default:
		panic(fmt.Sprint("something is seriously wrong with this arg of type CurrentOfUpdates:",
			c))
	}
}

type UnableToParseVersionNumber struct {
	CurrentOrUpdates CurrentOrUpdates
	What             string
	Err              error
}

func (*UnableToParseVersionNumber) Level() Level {
	return Warn
}

func (u *UnableToParseVersionNumber) Content() string {
	return fmt.Sprint(
		"Unable to parse", u.CurrentOrUpdates.String(),
		"version number: ", u.What)
}

type CurrentAndLatest struct {
	Current float32
	Latest  float32
}

type ComparisonOfVersioning int8

const (
	VeryOutOfDate ComparisonOfVersioning = -2
	ABitOutOfDate ComparisonOfVersioning = -1
	BeingUpToDate ComparisonOfVersioning = 0
	ADevelVersion ComparisonOfVersioning = 1
)

func (cnl *CurrentAndLatest) Compare() ComparisonOfVersioning {
	if cnl.Current < cnl.Latest {
		if math.Ceil(float64(cnl.Current)) < math.Ceil(float64(cnl.Latest)) {
			return VeryOutOfDate
		}
		return ABitOutOfDate
	}
	if cnl.Current > cnl.Latest {
		return ADevelVersion
	}
	return BeingUpToDate
}

func (cnl *CurrentAndLatest) Level() Level {
	if cnl.Compare() <= VeryOutOfDate {
		return Warn
	} else {
		return Info
	}
}

func (cnl *CurrentAndLatest) Content() string {
	var bef, mid, aft string
	switch cnl.Compare() {
	case VeryOutOfDate:
		bef = "KatWeb is very out of date"
		mid = "≪" //"≫"
		aft = "Please update to the latest version as soon as possible."
	case ABitOutOfDate:
		bef = "KatWeb is a bit out of date"
		mid = "<" //">"
		aft = "Using the latest version is recommended."
	case BeingUpToDate:
		bef = "KatWeb is up to date"
		mid = "="
		aft = "."
	case ADevelVersion:
		bef = "Running a development version of KatWeb"
		mid = ">" //"<"
		aft = "is not recommended."
	}
	return fmt.Sprint(
		bef, " (", cnl.Current, mid, cnl.Latest, ") ", aft)
}
