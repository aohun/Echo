//go:build nogui && echo

package settings

func init() { migrated = migrateEcho }

// migrateEcho runs after the files of an install that predates the name
// have been carried over from magpie's.
//
// The gateway's port does not carry over: it is the one setting that
// says which product a computer is running, and magpie's choice of one
// would put Echo on magpie's port — the port magpie's own app serves, so
// the two would hand the gateway over to whichever started first, and
// Echo would serve magpie's providers and routes.
//
// Every other setting does carry over: the keys, the providers, the
// agents' models, the accounts. Only the port is Echo's own.
func migrateEcho() {
	cur := Load()
	if cur.Port == 0 || cur.Port == DefaultPort {
		return
	}
	cur.Port = 0 // 0 is the default: Echo's own port
	_ = Save(cur)
}
