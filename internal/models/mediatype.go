package models

// Media type integer enum — stored in the cache database.
const (
	MediaTypeMusic  = 0
	MediaTypeMovies = 1
	MediaTypeShows  = 2
	MediaTypeImages = 3
)

// MediaTypeToInt converts a string media type to its integer representation.
var MediaTypeToInt = map[string]int{
	"music":  MediaTypeMusic,
	"movies": MediaTypeMovies,
	"shows":  MediaTypeShows,
	"images": MediaTypeImages,
}

// MediaTypeFromInt converts an integer media type back to its string representation.
var MediaTypeFromInt = map[int]string{
	MediaTypeMusic:  "music",
	MediaTypeMovies: "movies",
	MediaTypeShows:  "shows",
	MediaTypeImages: "images",
}
