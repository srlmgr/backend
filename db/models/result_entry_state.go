package models

const ResultEntryStateIgnore = "ignore"

func IsResultEntryIgnored(entry *ResultEntry) bool {
	return entry != nil && entry.State == ResultEntryStateIgnore
}
