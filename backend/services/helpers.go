package services

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseObjectID(id string) (primitive.ObjectID, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		// Malformed IDs can't match any record; report them as not found.
		return primitive.NilObjectID, notFoundf("invalid ID %q", id)
	}
	return objID, nil
}

func parseDate(dateStr string) (time.Time, error) {
	// Try multiple formats
	formats := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"02-Jan-2006",
		"02 Jan 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, invalidf("invalid date format: %s", dateStr)
}
