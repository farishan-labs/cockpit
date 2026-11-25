import Cocoa

// Minimal menubar launcher: a plain white circle that opens the cockpit.
// Left click opens the cockpit; right click shows the menu.
let COCKPIT_URL = "https://cockpit.local/"

final class CockpitIcon: NSObject {
    let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
    let menu = NSMenu()

    override init() {
        super.init()
        if let button = statusItem.button {
            if let img = NSImage(systemSymbolName: "circle.fill", accessibilityDescription: "Cockpit") {
                img.isTemplate = true
                button.image = img
            } else {
                button.title = "●"
            }
            button.toolTip = "Cockpit"
            button.target = self
            button.action = #selector(handleClick)
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }
        menu.addItem(NSMenuItem(title: "Open Cockpit", action: #selector(openCockpit), keyEquivalent: "o"))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(NSMenuItem(title: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        for item in menu.items { item.target = self }
    }

    @objc func handleClick() {
        if let event = NSApp.currentEvent, event.type == .rightMouseUp {
            statusItem.menu = menu
            statusItem.button?.performClick(nil)
            statusItem.menu = nil
        } else {
            openCockpit()
        }
    }

    @objc func openCockpit() {
        if let url = URL(string: COCKPIT_URL) {
            NSWorkspace.shared.open(url)
        }
    }
}

let app = NSApplication.shared
let icon = CockpitIcon()
app.setActivationPolicy(.accessory)
app.run()