// Package extensions defines the file extension whitelist and known
// media-root subfolder names for each media type.
package extensions

// ByType maps a media type name to the set of file extensions it owns.
// Extensions are lowercase and include the leading dot.
var ByType = map[string]map[string]bool{
	"music": {
		".mp3": true, ".flac": true, ".aac": true, ".ogg": true,
		".wav": true, ".wma": true, ".m4a": true, ".opus": true,
		".aiff": true, ".ape": true, ".wv": true,
	},
	"images": {
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".webp": true, ".tiff": true, ".tif": true,
	},
	"movies": {
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".wmv": true, ".m4v": true, ".flv": true, ".webm": true,
		".ts": true, ".m2ts": true, ".mpg": true, ".mpeg": true,
	},
	"shows": {
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".wmv": true, ".m4v": true, ".flv": true, ".webm": true,
		".ts": true, ".m2ts": true, ".mpg": true, ".mpeg": true,
	},
}

// SubfolderNames maps a media type to the ordered list of directory names
// the scanner looks for when resolving a type root in broad-scan mode.
// First match wins.
var SubfolderNames = map[string][]string{
	"movies": {"Movies", "movies", "Movie", "movie"},
	"music":  {"Music", "music"},
	"shows":  {"Shows", "shows", "TV", "tv", "TV Shows", "Series", "series"},
	"images": {"Images", "images", "Photos", "photos", "Pictures", "pictures"},
}

// Match returns true if the given lowercase extension belongs to mediaType.
func Match(mediaType, ext string) bool {
	exts, ok := ByType[mediaType]
	if !ok {
		return false
	}
	return exts[ext]
}
