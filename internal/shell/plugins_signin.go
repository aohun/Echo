//go:build nogui && echo

package shell

// A plugin's subscription signed in to, as magpie's page does it: the
// sign-in is the subscription's own (app.js's pluginSignIn), so the page
// goes to Providers and starts it there, with the account landing in the list
// as any subscription's does.

// plugSignInOpen starts the sign-in for the subscription a plugin's row
// offers: the Providers page, and the sign-in the site's own.
func (a *App) plugSignInOpen(sub PluginSubRow) {
	if sub.ID == "" {
		return
	}
	a.plugDetail = nil
	a.page = PageProviders
	if a.win != nil {
		go a.startSignIn(sub.ID)
		return
	}
	a.startSignIn(sub.ID)
}
