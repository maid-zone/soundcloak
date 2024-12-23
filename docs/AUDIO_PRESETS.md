| Name | Container  | Codec | Observed bitrate | Note                                                                                   |
| ------ | ------------ | ------- | ------------------ | ---------------------------------------------------------------------------------------- |
| Best |            |       |                  | Prefer AAC over Opus over MPEG. Not supported for HLS player (use AAC for same effect) |
| AAC  | mp4 (m4a)  | AAC   | 160kbps          | Rarely available. Falls back to MPEG if unavailable                                    |
| Opus | ogg        | Opus  | 72kbps           | Usually available. Falls back to MPEG if unavailable. Not supported for HLS player     |
| MP3  | mpeg (mp3) | MP3   | 128kbps          | Always available. Good for compatibility                                               |
