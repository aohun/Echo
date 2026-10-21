//go:build nogui && echo

package proc

func init() {
	// Echo's own name is magpie's, for the gateway's take-over and the
	// port guard: both ask whether the program holding a port is one of
	// ours before they touch it. Left out, Echo's own binary would be
	// read as some other program's, and the Gateway page's take-over
	// would say "the port is another program's" about Echo itself.
	//
	// It is matched only inside an app bundle, because the shell's own
	// echo is called the same thing: /bin/echo is not ours, and the
	// take-over must never end it.
	magpieBundleNames = append(magpieBundleNames, "echo")
}
