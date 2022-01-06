package output

import (
	"os"
)

func Example_writer_msgs() {
	w := NewColoredWriter(os.Stdout)

	w.Accent("Accent")
	w.Primary("Primary")
	w.Error("Error")

	// Output:
	// Accent
	// Primary
	// Error
}
