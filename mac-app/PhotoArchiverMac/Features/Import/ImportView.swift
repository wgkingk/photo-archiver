import SwiftUI
import AppKit

struct ImportView: View {
    @EnvironmentObject private var draft: ImportDraftStore
    @EnvironmentObject private var task: ImportTaskStore

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("导入")
                .font(.title2)
                .bold()

            HStack {
                TextField("来源目录", text: sourceRoot)
                Button("选择") { pickSourceDirectory() }
                Button("扫描") { Task { await scan() } }
                    .disabled(draft.sourceRoot.isEmpty || task.loading)
            }

            HStack {
                TextField("目标目录", text: destRoot)
                Button("选择") { pickDestDirectory() }
            }

            HStack {
                Picker("校验", selection: verifyMode) {
                    Text("size").tag("size")
                    Text("hash").tag("hash")
                }
                .pickerStyle(.segmented)
                Toggle("Dry Run", isOn: dryRun)
            }

            HStack {
                Button("开始导入") { Task { await startImport() } }
                    .disabled(draft.sourceRoot.isEmpty || draft.destRoot.isEmpty || task.loading)
                Button(task.cancelling ? "停止中..." : "停止任务") {
                    Task { await task.stopCurrentTask() }
                }
                .disabled(!task.canStop)
                if task.loading {
                    ProgressView()
                }
            }

            if !task.scanText.isEmpty {
                GroupBox("扫描结果") { Text(task.scanText).frame(maxWidth: .infinity, alignment: .leading) }
            }
            if !task.resultText.isEmpty {
                GroupBox("导入结果") { Text(task.resultText).frame(maxWidth: .infinity, alignment: .leading) }
            }

            if !task.currentJobID.isEmpty {
                GroupBox("任务进度") {
                    VStack(alignment: .leading, spacing: 8) {
                        Text("任务ID: \(task.currentJobID)")
                            .font(.caption)
                            .foregroundStyle(.secondary)

                        ProgressView(value: progressValue, total: progressTotal)

                        HStack(spacing: 12) {
                            Text("状态: \(task.jobStatus)")
                            Text("成功: \(task.successCount)")
                            Text("跳过: \(task.skippedCount)")
                            Text("失败: \(task.failedCount)")
                        }
                        .font(.caption)
                        .foregroundStyle(.secondary)

                        Text("预计剩余: \(task.etaText)")
                            .font(.caption)
                            .foregroundStyle(.secondary)

                        if task.consecutivePollFailures > 0 {
                            Text("网络波动，正在重试（第 \(task.consecutivePollFailures) 次）")
                                .font(.caption)
                                .foregroundStyle(.orange)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }

            Spacer()
        }
        .padding()
    }

    private func scan() async {
        await task.scan(sourceRoot: draft.sourceRoot)
    }

    private func startImport() async {
        await task.startImport(sourceRoot: draft.sourceRoot, destRoot: draft.destRoot, dryRun: draft.dryRun, verifyMode: draft.verifyMode)
    }

    private var progressValue: Double {
        Double(task.successCount + task.skippedCount + task.failedCount)
    }

    private var progressTotal: Double {
        task.totalCount > 0 ? Double(task.totalCount) : 1
    }

    private func pickSourceDirectory() {
        if let p = selectDirectory() {
            draft.sourceRoot = p
        }
    }

    private func pickDestDirectory() {
        if let p = selectDirectory() {
            draft.destRoot = p
        }
    }

    private func selectDirectory() -> String? {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.allowsMultipleSelection = false
        panel.prompt = "选择"
        let response = panel.runModal()
        guard response == .OK else { return nil }
        return panel.url?.path
    }

    private var sourceRoot: Binding<String> { $draft.sourceRoot }
    private var destRoot: Binding<String> { $draft.destRoot }
    private var verifyMode: Binding<String> { $draft.verifyMode }
    private var dryRun: Binding<Bool> { $draft.dryRun }
}
