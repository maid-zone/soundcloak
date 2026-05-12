| Name            | Container  | Observed bitrate | Note                                                                               |
| --------------- | ---------- | ---------------- | ---------------------------------------------------------------------------------- |
| `"aac"`         | mp4 (m4a)  | 160kbps          | Mostly available. Falls back to MPEG if unavailable                                |
| `"mpeg"`        | mpeg (mp3) | 128kbps          | Always available for now. Will probably be removed and replaced by low quality AAC |
| `"aac_lq"`      | mp4 (m4a)  | 96kbps           | Most recently added. Good for low bandwidth users (?)                              |
