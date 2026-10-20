# Echo — every agent's model, one place.
#
# The tap is this repository:
#
#   brew tap aohun/echo https://github.com/aohun/Echo.git
#   brew install --cask aohun/echo/echo
#   brew upgrade --cask aohun/echo/echo
#
# The release workflow rewrites the version and the hashes below when it
# publishes, so they always name the assets on that release.
cask "echo" do
  version "0.0.1"

  arch arm: "arm64"
  arch intel: "x86_64"

  url "https://github.com/aohun/Echo/releases/download/v#{version}/Echo-#{version}-#{arch}.dmg"
  sha256 arm: "1bd12cddd6a0f078be7cd25d99b51bf3c129cecd410b0fe8f2bbd7dc169d978c",
        intel: "6854e58a05bb4d56da5a52971f08f0a4eb5d419c14af8862438e138280703df2"

  name "Echo"
  desc "Every agent's model. One place."
  homepage "https://github.com/aohun/Echo"

  depends_on macos: ">= :ventura"

  app "Echo.app"

  # a menu bar app: it keeps no window of its own, and the gateway it runs
  # is in the menu bar icon
  zap trash: [
    "~/Library/Application Support/Echo",
    "~/Library/Caches/echo",
    "~/Library/Preferences/ai.echo.app.plist",
    "~/Library/Logs/Echo",
    "~/.config/echo",
    "~/.cache/echo",
  ]
end
