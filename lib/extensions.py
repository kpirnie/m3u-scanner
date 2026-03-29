"""Media type → file extension mapping and known subfolder names."""

EXTENSIONS: dict[str, frozenset[str]] = {
    "music": frozenset({
        ".mp3", ".flac", ".aac", ".ogg", ".wav", ".wma",
        ".m4a", ".opus", ".aiff", ".ape", ".wv",
    }),
    "images": frozenset({
        ".jpg", ".jpeg", ".png", ".gif", ".bmp",
        ".webp", ".tiff", ".tif",
    }),
    "movies": frozenset({
        ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".m4v",
        ".flv", ".webm", ".ts", ".m2ts", ".mpg", ".mpeg",
    }),
    "shows": frozenset({
        ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".m4v",
        ".flv", ".webm", ".ts", ".m2ts", ".mpg", ".mpeg",
    }),
}

# Recognized subfolder names per type, used in broad-mode scanning.
# The scanner walks root looking for any folder whose name appears in this list.
TYPE_SUBFOLDERS: dict[str, list[str]] = {
    "movies": ["Movies", "movies", "Movie", "movie"],
    "music":  ["Music", "music"],
    "shows":  ["Shows", "shows", "TV", "tv", "TV Shows", "Series", "series"],
    "images": ["Images", "images", "Photos", "photos", "Pictures", "pictures"],
}
