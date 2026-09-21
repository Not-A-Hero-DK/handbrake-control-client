import AppKit

private let appVersion = (Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String) ?? "0.1.0"

private struct AgentConfig: Codable {
    struct Agent: Codable {
        var id: String
        var displayName: String
        var heartbeatSeconds: Int
        var socketPath: String
    }

    struct Server: Codable {
        var url: String
        var enrollmentToken: String
    }

    struct Worker: Codable {
        var workDirectory: String
        var enabled: Bool
    }

    struct MountProfile: Codable {
        var name: String
        var shareURL: String
        var mountPoint: String
        var credentialKeychainService: String
    }

    struct Power: Codable {
        var allowClosedLidWork: Bool
        var requireACPower: Bool
        var stopIfBatteryBelowPercent: Int
        var restoreSleepOnIdle: Bool
    }

    var agent: Agent
    var server: Server
    var worker: Worker
    var mountProfiles: [MountProfile]
    var power: Power
}

@main
struct HandBrakeControlApplication {
    private static let delegate = AppDelegate()

    static func main() {
        let application = NSApplication.shared
        application.delegate = delegate
        application.run()
    }
}

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var statusItem: NSStatusItem!
    private var stateItem: NSMenuItem!
    private var agentProcess: Process?
    private var refreshTimer: Timer?
    private var configURL: URL?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
        configURL = prepareLocalConfig()

        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        guard let button = statusItem.button else { return }
        button.image = NSImage(systemSymbolName: "film", accessibilityDescription: "HandBrake Control")
        button.imagePosition = .imageLeading
        button.toolTip = "HandBrake Control: starting"

        let menu = NSMenu()
        stateItem = NSMenuItem(title: "Status: Starting…", action: nil, keyEquivalent: "")
        stateItem.isEnabled = false
        menu.addItem(stateItem)
        menu.addItem(withTitle: "Refresh status", action: #selector(refreshStatus), keyEquivalent: "r")
        menu.addItem(.separator())
        menu.addItem(withTitle: "Settings…", action: #selector(showSettings), keyEquivalent: ",")
        menu.addItem(.separator())
        menu.addItem(withTitle: "Start worker", action: #selector(startWorker), keyEquivalent: "")
        menu.addItem(withTitle: "Stop and requeue", action: #selector(stopAndRequeue), keyEquivalent: "")
        menu.addItem(withTitle: "Drain after current job", action: #selector(drainWorker), keyEquivalent: "")
        menu.addItem(.separator())
        menu.addItem(withTitle: "About HandBrake Control", action: #selector(showAbout), keyEquivalent: "")
        let versionItem = menu.addItem(withTitle: "Version \(appVersion)", action: nil, keyEquivalent: "")
        versionItem.isEnabled = false
        menu.addItem(.separator())
        menu.addItem(withTitle: "Quit HandBrake Control", action: #selector(quit), keyEquivalent: "q")
        statusItem.menu = menu

        startAgent()
        refreshStatus()
        refreshTimer = Timer.scheduledTimer(withTimeInterval: 5, repeats: true) { [weak self] _ in
            self?.refreshStatus()
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        refreshTimer?.invalidate()
        agentProcess?.terminate()
    }

    @objc private func startWorker() { showUnavailable("Start worker") }
    @objc private func stopAndRequeue() { showUnavailable("Stop and requeue") }
    @objc private func drainWorker() { showUnavailable("Drain after current job") }

    @objc private func quit() { NSApp.terminate(nil) }

    @objc private func showAbout() {
        let alert = NSAlert()
        alert.messageText = "HandBrake Control \(appVersion)"
        alert.informativeText = "Created by Mads for Not A Hero.\n\nA local readiness client for the future HandBrake Control server."
        alert.addButton(withTitle: "OK")
        alert.runModal()
    }

    @objc private func showSettings() {
        guard let configURL, var config = try? loadConfig(at: configURL) else {
            setState("Could not load local settings")
            return
        }

        let agentName = textField(config.agent.displayName)
        let serverURL = textField(config.server.url)
        let mountShare = textField(config.mountProfiles.first?.shareURL ?? "")
        let workerEnabled = checkbox("Enable this worker when conversion support is available", config.worker.enabled)
        let closedLid = checkbox("Allow closed-lid work", config.power.allowClosedLidWork)
        let requireAC = checkbox("Require AC power", config.power.requireACPower)
        let restoreSleep = checkbox("Restore normal sleep when idle", config.power.restoreSleepOnIdle)
        let minimumBattery = NSTextField(string: String(config.power.stopIfBatteryBelowPercent))
        minimumBattery.alignment = .right
        minimumBattery.frame.size.width = 52

        let batteryRow = NSStackView(views: [
            NSTextField(labelWithString: "Stop work below battery percentage:"), minimumBattery,
            NSTextField(labelWithString: "%")
        ])
        batteryRow.orientation = .horizontal
        batteryRow.spacing = 8

        let connectionStack = NSStackView(views: [
            labelledField("Agent name", agentName),
            labelledField("Server URL", serverURL),
            labelledField("SMB share URL", mountShare),
            fixedValue("HandBrakeCLI", "Detected automatically from macOS or Homebrew"),
            fixedValue("Local conversion data", "~/handbrake-control"),
            fixedValue("Finder mount location", "~/Volumes/HandBrake Control Media")
        ])
        connectionStack.orientation = .vertical
        connectionStack.alignment = .leading
        connectionStack.spacing = 10
        connectionStack.edgeInsets = NSEdgeInsets(top: 8, left: 8, bottom: 8, right: 8)

        let workerStack = NSStackView(views: [workerEnabled, closedLid, requireAC, restoreSleep, batteryRow])
        workerStack.orientation = .vertical
        workerStack.alignment = .leading
        workerStack.spacing = 10
        workerStack.edgeInsets = NSEdgeInsets(top: 8, left: 8, bottom: 8, right: 8)

        // Adjust these two values to change the Settings window's width and height.
        let settingsContentSize = NSSize(width: 720, height: 300)
        let tabView = NSTabView(frame: NSRect(origin: .zero, size: settingsContentSize))
        tabView.addTabViewItem(tab(title: "Connection & files", view: paddedContainer(for: connectionStack, size: NSSize(width: 700, height: 260))))
        tabView.addTabViewItem(tab(title: "Worker & power", view: paddedContainer(for: workerStack, size: NSSize(width: 700, height: 260))))

        let alert = NSAlert()
        alert.messageText = "Worker settings"
        alert.informativeText = "Preferences are stored locally. Conversion, SMB mounting, server connection, and power control are planned milestones."
        alert.accessoryView = tabView
        alert.addButton(withTitle: "Save")
        alert.addButton(withTitle: "Cancel")
        alert.layout()
        alert.window.styleMask.formUnion([.closable, .resizable])
        alert.window.standardWindowButton(.closeButton)?.isEnabled = true
        alert.window.minSize = NSSize(width: 790, height: 470)
        alert.window.setContentSize(NSSize(width: 790, height: 500))
        guard alert.runModal() == .alertFirstButtonReturn else { return }

        guard let battery = Int(minimumBattery.stringValue), (0...100).contains(battery) else {
            setState("Battery percentage must be between 0 and 100")
            return
        }
        config.worker.enabled = workerEnabled.state == .on
        config.agent.displayName = agentName.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        config.server.url = serverURL.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let configuredShare = mountShare.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        if !configuredShare.isEmpty {
            if config.mountProfiles.isEmpty {
                config.mountProfiles.append(.init(name: "media", shareURL: configuredShare, mountPoint: "${HOME}/Volumes/HandBrake Control Media", credentialKeychainService: "handbrake-control.media"))
            } else {
                config.mountProfiles[0].shareURL = configuredShare
            }
        }
        config.power.allowClosedLidWork = closedLid.state == .on
        config.power.requireACPower = requireAC.state == .on
        config.power.restoreSleepOnIdle = restoreSleep.state == .on
        config.power.stopIfBatteryBelowPercent = battery

        do {
            try saveConfig(config, to: configURL)
            restartAgent()
            setState("Settings saved")
        } catch {
            setState("Could not save settings")
        }
    }

    @objc private func refreshStatus() {
        guard agentProcess?.isRunning == true else {
            setState("Agent is not running")
            return
        }
        guard let configURL else { return }
        let agentURL = agentExecutableURL()

        DispatchQueue.global(qos: .utility).async { [weak self] in
            let process = Process()
            let output = Pipe()
            process.executableURL = agentURL
            process.arguments = ["-config", configURL.path, "status"]
            process.standardOutput = output
            process.standardError = Pipe()
            do {
                try process.run()
                process.waitUntilExit()
                let data = output.fileHandleForReading.readDataToEndOfFile()
                guard process.terminationStatus == 0,
                      let object = try JSONSerialization.jsonObject(with: data) as? [String: Any] else {
                    throw NSError(domain: "HandBrakeControl", code: 1)
                }
                let handBrakeReady = object["handbrakeReady"] as? Bool ?? false
                let mounts = object["mounts"] as? [[String: Any]] ?? []
                let mounted = !mounts.isEmpty && mounts.allSatisfy { $0["mounted"] as? Bool == true }
                let message = handBrakeReady
                    ? "Agent running · Mount \(mounted ? "ready" : "not mounted")"
                    : "Agent running · HandBrake setup needs attention"
                DispatchQueue.main.async { self?.setState(message) }
            } catch {
                DispatchQueue.main.async { self?.setState("Agent status unavailable") }
            }
        }
    }

    private func startAgent() {
        guard let configURL else {
            setState("Agent bundle is incomplete")
            return
        }
        let agentURL = agentExecutableURL()
        let process = Process()
        process.executableURL = agentURL
        process.arguments = ["-config", configURL.path, "daemon"]
        process.standardOutput = Pipe()
        process.standardError = Pipe()
        do {
            try process.run()
            agentProcess = process
        } catch {
            setState("Could not start agent")
        }
    }

    private func restartAgent() {
        agentProcess?.terminate()
        agentProcess = nil
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) { [weak self] in
            self?.startAgent()
            self?.refreshStatus()
        }
    }

    private func prepareLocalConfig() -> URL? {
        guard let templateURL = Bundle.main.url(forResource: "agent", withExtension: "json"),
              let supportDirectory = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first else {
            return nil
        }
        let directory = supportDirectory.appendingPathComponent("HandBrake Control", isDirectory: true)
        let localConfig = directory.appendingPathComponent("agent.json")
        do {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            if !FileManager.default.fileExists(atPath: localConfig.path) {
                try FileManager.default.copyItem(at: templateURL, to: localConfig)
            }
            return localConfig
        } catch {
            return nil
        }
    }

    private func agentExecutableURL() -> URL {
        Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/hb-agent")
    }

    private func loadConfig(at url: URL) throws -> AgentConfig {
        try JSONDecoder().decode(AgentConfig.self, from: Data(contentsOf: url))
    }

    private func saveConfig(_ config: AgentConfig, to url: URL) throws {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        try encoder.encode(config).write(to: url, options: .atomic)
    }

    private func checkbox(_ title: String, _ isOn: Bool) -> NSButton {
        let button = NSButton(checkboxWithTitle: title, target: nil, action: nil)
        button.state = isOn ? .on : .off
        return button
    }

    private func textField(_ value: String) -> NSTextField {
        let field = NSTextField(string: value)
        field.widthAnchor.constraint(equalToConstant: 440).isActive = true
        field.setContentHuggingPriority(.defaultLow, for: .horizontal)
        field.setContentCompressionResistancePriority(.required, for: .horizontal)
        return field
    }

    private func labelledField(_ label: String, _ field: NSTextField) -> NSStackView {
        let labelField = NSTextField(labelWithString: "\(label):")
        labelField.widthAnchor.constraint(equalToConstant: 160).isActive = true
        let row = NSStackView(views: [labelField, field])
        row.orientation = .horizontal
        row.alignment = .firstBaseline
        row.spacing = 8
        return row
    }

    private func fixedValue(_ label: String, _ value: String) -> NSStackView {
        let valueLabel = NSTextField(labelWithString: value)
        valueLabel.textColor = .secondaryLabelColor
        return labelledField(label, valueLabel)
    }

    private func paddedContainer(for stack: NSStackView, size: NSSize) -> NSView {
        let container = NSView(frame: NSRect(origin: .zero, size: size))
        stack.translatesAutoresizingMaskIntoConstraints = false
        container.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 14),
            stack.trailingAnchor.constraint(lessThanOrEqualTo: container.trailingAnchor, constant: -14),
            stack.topAnchor.constraint(equalTo: container.topAnchor, constant: 12)
        ])
        return container
    }

    private func tab(title: String, view: NSView) -> NSTabViewItem {
        let item = NSTabViewItem(identifier: title)
        item.label = title
        item.view = view
        return item
    }

    private func setState(_ message: String) {
        stateItem.title = "Status: \(message)"
        statusItem.button?.toolTip = "HandBrake Control: \(message)"
    }

    private func showUnavailable(_ action: String) {
        setState("\(action) will be available with the conversion worker")
    }
}
