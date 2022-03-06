package extra

import (
	"image/jpeg"
	"os"
	"testing"
)

func TestResizeImage(t *testing.T) {
	src, err := os.Open("./testdata/src.png")
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	out, err := os.Create("./testdata/out.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	if err := ResizeImage("src.png", src, out, 1600.0); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("./testdata/out.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	} else if img.Bounds().Max.X > 1600 {
		t.Errorf("expected resized img to have <= 1600 wide got %d", img.Bounds().Max.X)
	}
}
