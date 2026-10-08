package instance

import (
	"testing"
	"time"
)

func TestSecondStartShowsTheFirst(t *testing.T) {
	dir := t.TempDir()

	first, ok := Acquire(dir, nil)
	if !ok {
		t.Fatal("the first start must run")
	}
	shown := make(chan []string, 1)
	first.Serve(func(files []string) { shown <- files })

	if _, ok := Acquire(dir, []string{"/a b/c.png", "/d.jpg"}); ok {
		t.Fatal("the second start must quit while the first runs")
	}
	select {
	case files := <-shown:
		if len(files) != 2 || files[0] != "/a b/c.png" || files[1] != "/d.jpg" {
			t.Fatalf("files passed: %q", files)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the first copy was not asked to show itself")
	}

	first.Close()
	again, ok := Acquire(dir, nil)
	if !ok {
		t.Fatal("a start after the first has quit must run")
	}
	again.Close()
}
