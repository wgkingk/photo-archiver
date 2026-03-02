import SwiftUI

enum SidebarItem: String, CaseIterable, Identifiable {
    case `import` = "导入"
    case jobs = "任务"
    case settings = "设置"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .import:
            return "square.and.arrow.down"
        case .jobs:
            return "clock.arrow.circlepath"
        case .settings:
            return "gearshape"
        }
    }
}

struct ContentView: View {
    @EnvironmentObject private var backend: BackendProcessManager
    @State private var selection: SidebarItem = .import

    var body: some View {
        NavigationSplitView {
            List {
                ForEach(SidebarItem.allCases) { item in
                    Button {
                        selection = item
                    } label: {
                        HStack(spacing: 10) {
                            Image(systemName: item.icon)
                                .frame(width: 18)
                            Text(item.rawValue)
                            Spacer()
                        }
                        .padding(.vertical, 6)
                        .padding(.horizontal, 8)
                        .background(selection == item ? Color.accentColor.opacity(0.16) : Color.clear)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                    }
                    .buttonStyle(.plain)
                }
            }
            .listStyle(.sidebar)
            .navigationTitle("照片归档")
        } detail: {
            switch selection {
            case .settings:
                SettingsView()
            case .import, .jobs:
                if backend.isReady {
                    if selection == .import {
                        ImportView()
                    } else {
                        JobsView()
                    }
                } else {
                    VStack(spacing: 12) {
                        ProgressView()
                        Text(backend.statusText)
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
        }
        .navigationSplitViewStyle(.balanced)
    }
}

private struct SettingsView: View {
    @EnvironmentObject private var backend: BackendProcessManager

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                GroupBox("应用") {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("应用名称：照片归档")
                        Text("版本：开发中")
                            .foregroundStyle(.secondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }

                GroupBox("后端状态") {
                    VStack(alignment: .leading, spacing: 10) {
                        keyValue("运行状态", backend.isReady ? "运行中" : "未就绪", valueColor: backend.isReady ? .green : .orange)
                        keyValue("地址", backend.baseURL.absoluteString)
                        keyValue("端口", backend.currentPort.map { String($0) } ?? "-")
                        keyValue("PID", backend.backendPID.map { String($0) } ?? "-")
                        keyValue("状态", backend.statusText)

                        HStack {
                            Button("重启后端") {
                                Task { await backend.restart() }
                            }
                            .keyboardShortcut("r", modifiers: [.command, .shift])

                            Button("停止后端") {
                                backend.stop()
                            }
                            .disabled(!backend.isReady)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .frame(maxWidth: .infinity, alignment: .topLeading)
            .padding()
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    private func keyValue(_ key: String, _ value: String, valueColor: Color = .secondary) -> some View {
        HStack {
            Text(key)
            Spacer()
            Text(value)
                .foregroundStyle(valueColor)
        }
    }
}
