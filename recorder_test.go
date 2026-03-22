package browse

import "testing"

func TestRecorderOptionsStruct(t *testing.T) {
	opts := RecorderOptions{
		Format:    "png",
		Quality:   90,
		MaxWidth:  1280,
		MaxHeight: 720,
	}

	if opts.Format != "png" {
		t.Errorf("Format = %q, want %q", opts.Format, "png")
	}
	if opts.Quality != 90 {
		t.Errorf("Quality = %d, want 90", opts.Quality)
	}
	if opts.MaxWidth != 1280 {
		t.Errorf("MaxWidth = %d, want 1280", opts.MaxWidth)
	}
	if opts.MaxHeight != 720 {
		t.Errorf("MaxHeight = %d, want 720", opts.MaxHeight)
	}
}

func TestRecorderOptionsJPEGDefaults(t *testing.T) {
	opts := RecorderOptions{
		Format:  "jpeg",
		Quality: 80,
	}
	if opts.Format != "jpeg" {
		t.Errorf("Format = %q, want jpeg", opts.Format)
	}
}
