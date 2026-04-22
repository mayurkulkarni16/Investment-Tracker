package services

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func parseObjectID(id string) (primitive.ObjectID, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("invalid ID format: %w", err)
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
	return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
}
