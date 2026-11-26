import Cocoa

// Minimal menubar launcher: a circle that opens the cockpit.
// Green/white when the cockpit responds, red when it is unreachable (e.g. 502).
let COCKPIT_URL = "https://cockpit.local/"
let HEALTH_URL = "http://127.0.0.1:7777/"

final class CockpitIcon: NSObject {
    let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
    let menu = NSMenu()
    var timer: Timer?

    override init() {
        super.init()
        if let button = statusItem.button {
            button.target = self
            button.action = #selector(handleClick)
            button.sendAction(on: [.leftMouseUp, .rightMouseUp])
        }
        menu.addItem(NSMenuItem(title: "Open Cockpit", action: #selector(openCockpit), keyEquivalent: "o"))
        menu.addItem(NSMenuItem.separator())
        menu.addItem(NSMenuItem(title: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        for item in menu.items { item.target = self }

        setHealthy(true)
        checkHealth()
        timer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in self?.checkHealth() }
    }

    func makeCircle(color: NSColor) -> NSImage {
        let size = NSSize(width: 14, height: 14)
        let image = NSImage(size: size)
        image.lockFocus()
        color.setFill()
        NSBezierPath(ovalIn: NSRect(origin: .zero, size: size)).fill()
        image.unlockFocus()
        image.isTemplate = false
        return image
    }

    func setHealthy(_ healthy: Bool) {
        if let button = statusItem.button {
            if healthy {
                if let img = NSImage(systemSymbolName: "circle.fill", accessibilityDescription: "Cockpit") {
                    img.isTemplate = true
                    button.image = img
                }
            } else {
                button.image = makeCircle(color: .systemRed)
            }
            button.toolTip = healthy ? "Cockpit (healthy)" : "Cockpit (unreachable)"
        }
    }

    func checkHealth() {
        guard let url = URL(string: HEALTH_URL) else { return }
        var req = URLRequest(url: url, timeoutInterval: 5)
        req.httpMethod = "GET"
        let task = URLSession.shared.dataTask(with: req) { [weak self] _, response, error in
            let healthy = (error == nil) && (200...399).contains((response as? HTTPURLResponse)?.statusCode ?? 0)
            DispatchQueue.main.async { self?.setHealthy(healthy) }
        }
        task.resume()
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