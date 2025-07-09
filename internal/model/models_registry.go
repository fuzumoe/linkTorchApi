package model

// AllModels contains instances of all database models for auto-migrations
var AllModels = []interface{}{
	&User{},
	&URL{},
	&AnalysisResult{},
	&Link{},
	&BlacklistedToken{},
}
