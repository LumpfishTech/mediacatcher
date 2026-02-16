package ytdlp

import (
	"strings"
	"testing"
)

func TestParseFormats(t *testing.T) {
	sampleOutput := `[youtube] Extracting URL: https://www.youtube.com/watch?v=XXXXXXXXXXX
ID  EXT   RESOLUTION  FPS │   FILESIZE   TBR PROTO │ VCODEC          VBR ACODEC      ABR ASR
───────────────────────────────────────────────────────────────────────────────────────────────
sb2 mhtml 48x27           │                  mhtml │ images
sb1 mhtml 80x45           │                  mhtml │ images
139 m4a   audio only      │    1.27MiB   48k https │ audio only          mp4a.40.5   48k 22050Hz
249 webm  audio only      │    1.30MiB   49k https │ audio only          opus        49k 48000Hz
250 webm  audio only      │    1.73MiB   65k https │ audio only          opus        65k 48000Hz
140 m4a   audio only      │    3.41MiB  128k https │ audio only          mp4a.40.2  128k 44100Hz
251 webm  audio only      │    3.44MiB  129k https │ audio only          opus       129k 48000Hz
17  3gp   176x144      6  │    1.19MiB   45k https │ mp4v.20.3       45k mp4a.40.2    0k 22050Hz
160 mp4   256x144     12  │    1.31MiB   49k https │ avc1.4d400b     49k video only
278 webm  256x144     12  │    1.39MiB   52k https │ vp9             52k video only
18  mp4   640x360     24  │    6.50MiB  244k https │ avc1.42001E    244k mp4a.40.2    0k 44100Hz
22  mp4   1280x720    24  │ ~ 28.84MiB 1084k https │ avc1.64001F   1084k mp4a.40.2    0k 44100Hz`

	formats := parseFormats(sampleOutput)

	if len(formats) == 0 {
		t.Fatal("Expected formats to be parsed, got empty list")
	}

	// Check first audio format
	found139 := false
	for _, f := range formats {
		if f.ID == "139" {
			found139 = true
			if f.Extension != "m4a" {
				t.Errorf("Expected extension m4a for format 139, got %s", f.Extension)
			}
			if !strings.Contains(f.Description, "139") {
				t.Errorf("Expected description to contain ID, got %s", f.Description)
			}
		}
	}

	if !found139 {
		t.Error("Expected to find format 139 in parsed formats")
	}

	// Check video format
	found22 := false
	for _, f := range formats {
		if f.ID == "22" {
			found22 = true
			if f.Extension != "mp4" {
				t.Errorf("Expected extension mp4 for format 22, got %s", f.Extension)
			}
			if f.Resolution != "1280x720" {
				t.Errorf("Expected resolution 1280x720 for format 22, got %s", f.Resolution)
			}
		}
	}

	if !found22 {
		t.Error("Expected to find format 22 in parsed formats")
	}
}

func TestParseFormatsEmpty(t *testing.T) {
	formats := parseFormats("")
	if len(formats) != 0 {
		t.Errorf("Expected empty formats list for empty input, got %d formats", len(formats))
	}
}

func TestParseFormatsNoHeader(t *testing.T) {
	// Output without proper header should return empty
	sampleOutput := `Some random text
139 m4a audio only
140 m4a audio only`

	formats := parseFormats(sampleOutput)
	if len(formats) != 0 {
		t.Errorf("Expected empty formats without header, got %d formats", len(formats))
	}
}
