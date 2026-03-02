import SwiftUI
import AppKit

@main
struct PhotoArchiverMacApp: App {
    @StateObject private var backend = BackendProcessManager()
    @StateObject private var importDraft = ImportDraftStore()
    @StateObject private var importTask = ImportTaskStore()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(backend)
                .environmentObject(importDraft)
                .environmentObject(importTask)
                .task {
                    await backend.start()
                }
                .onReceive(NotificationCenter.default.publisher(for: NSApplication.willTerminateNotification)) { _ in
                    backend.stop()
                }
        }
        .windowStyle(.automatic)
    }
}
