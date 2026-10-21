//go:build nogui && echo

package provider

func init() {
	// Echo registers the scheme its own bundle declares (echo://, in
	// mygo.json), so the links the system hands over — and the ones a
	// vendor writes for a user to click — are Echo's.
	//
	// Left at magpie's, Echo would register echo:// and read magpie://:
	// a link handed to the app would not be recognized, and a link
	// written for it would be read by magpie instead.
	Scheme = "echo"
}
