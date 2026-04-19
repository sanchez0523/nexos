# Recording the Nexos demo

The hero GIF on the README and landing page is the single biggest lever for
GitHub Stars. This guide produces a 30-second clip that demonstrates the
Auto-Discovery differentiator end-to-end.

## Target spec

| Property   | Value                         |
|------------|-------------------------------|
| Resolution | 1280 × 720                    |
| Frame rate | 30 fps                        |
| Duration   | 25–30 seconds                 |
| Output     | `docs/public/demo.gif` (<10 MB) + `docs/public/demo.mp4` |

GitHub caps inline images at 10 MB. GIFs that big are grainy; we keep the
MP4 too for the landing site where we can use `<video autoplay loop muted>`.

## Before recording

1. **Fresh dashboard state.** Clear localStorage in the browser so the grid
   starts empty. Run `./scripts/setup.sh` in a scratch directory with
   `admin` / `demo` as the admin credentials so the typed password isn't
   sensitive.
2. **Pre-loaded simulator.** Run 3 accounts via `./scripts/add-device.sh sim-1` etc.
   Don't start the simulator yet — it's part of the recording.
3. **Hide the cursor trail / notifications** — close chat apps, enable Do
   Not Disturb. Set the terminal prompt to something minimal (e.g.
   `PS1='$ '`).
4. **Full-screen browser** at 1280×720. Use Chrome's device toolbar to pin
   the viewport so the recording has consistent dimensions.

## Shot list (30s total)

| Time | Shot | Action |
|------|------|--------|
| 0:00–0:04 | Terminal | `./scripts/setup.sh` scrolls briefly, closes with ✔ success lines |
| 0:04–0:07 | Terminal | `docker compose up -d` → all services `healthy` |
| 0:07–0:10 | Browser  | Login page → sign in → empty dashboard ("Waiting for the first metric…") |
| 0:10–0:18 | Terminal | Start simulator. **Cut to dashboard**: cards appear one by one as topics arrive |
| 0:18–0:23 | Browser  | Drag a card to resize; drop. Cards rearrange smoothly |
| 0:23–0:28 | Browser  | Open `/alerts`. Fill the form. Save. Back to dashboard, threshold is exceeded, console shows webhook log |
| 0:28–0:30 | End card | "Nexos · MIT · github.com/OWNER/REPO" on black |

## Recording

### macOS (OBS + screen capture)

```bash
# install once
brew install --cask obs
brew install ffmpeg
```

In OBS:
1. Scene → Add → **Display Capture** (primary display)
2. Filters → Crop/Pad to 1280×720 region
3. Settings → Video → Output Resolution 1280×720, 30 fps
4. Settings → Output → Recording Format `mp4`, Encoder `Apple VT H.264 Hardware`

Hit **Start Recording**, run through the shot list, **Stop Recording**.

### Linux (wf-recorder / ffmpeg direct)

```bash
ffmpeg -video_size 1280x720 -framerate 30 -f x11grab -i :0.0+100,100 \
       -c:v libx264 -preset veryfast -crf 22 -pix_fmt yuv420p demo.mp4
```

## Post-process

### MP4 → optimized GIF

```bash
# 1. Extract palette for better colors
ffmpeg -i demo.mp4 -vf "fps=15,scale=960:-1:flags=lanczos,palettegen" palette.png

# 2. Use the palette to make a size-efficient gif
ffmpeg -i demo.mp4 -i palette.png -filter_complex \
  "fps=15,scale=960:-1:flags=lanczos[x];[x][1:v]paletteuse" \
  docs/public/demo.gif
```

Target <10 MB. If the GIF is larger, drop fps to 12 or scale to 800 px.

### Keep the MP4 too

```bash
ffmpeg -i demo.mp4 -vcodec h264 -acodec aac -movflags +faststart \
       docs/public/demo.mp4
```

## README integration

```markdown
<p align="center">
  <img src="docs/public/demo.gif" alt="Nexos demo — from MQTT publish to auto-generated dashboard card" width="720" />
</p>
```

GitHub Pages landing uses the MP4 with `<video>` for smoother playback and
smaller download.

## QA checklist before committing

- [ ] No personal info visible (terminal prompt, browser bookmarks, other tabs)
- [ ] No real passwords or JWTs on screen
- [ ] Cards visibly appear without manual refresh (the differentiator)
- [ ] Drag-and-drop is shown
- [ ] Alert fire is shown
- [ ] Duration 25–35s (strict — longer fails to hold attention on social)
- [ ] GIF under 10 MB
