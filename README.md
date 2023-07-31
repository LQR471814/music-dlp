## music-dlp

> a simple wrapper around [yt-dlp](https://github.com/yt-dlp/yt-dlp) that makes it easier to download music and populate it with metadata.

```
usage:
./music-dlp YOUTUBE_LINK or FILENAME
```

### features

it runs `yt-dlp` using the following flags.

```
yt-dlp --write-info-json -x --audio-format mp3 --sponsorblock-mark all -o %(title)s.%(ext)s
```

then it provides an interface for changing the output file's metadata.

you can also provide a filename instead of a youtube link to edit an mp3 file's metadata.
