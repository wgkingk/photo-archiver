import Foundation

@MainActor
final class ImportTaskStore: ObservableObject {
    @Published var scanText: String = ""
    @Published var resultText: String = ""
    @Published var loading: Bool = false

    @Published var currentJobID: String = ""
    @Published var jobStatus: String = ""
    @Published var totalCount: Int = 0
    @Published var successCount: Int = 0
    @Published var skippedCount: Int = 0
    @Published var failedCount: Int = 0
    @Published var consecutivePollFailures: Int = 0
    @Published var etaText: String = "--"
    @Published var cancelling: Bool = false

    private var pollingTask: Task<Void, Never>?
    private var progressSamples: [(time: Date, done: Int)] = []

    func scan(sourceRoot: String) async {
        loading = true
        defer { loading = false }
        do {
            let r = try await APIClient.shared.scan(sourceRoot: sourceRoot)
            scanText = "总数: \(r.totalCount), 总大小: \(r.totalBytes) bytes, EXIF命中: \(r.exifCount), 回退: \(r.fallbackCount)"
        } catch {
            scanText = "扫描失败: \(error.localizedDescription)"
        }
    }

    func startImport(sourceRoot: String, destRoot: String, dryRun: Bool, verifyMode: String) async {
        loading = true
        cancelling = false
        do {
            let accepted = try await APIClient.shared.startImportAsync(sourceRoot: sourceRoot, destRoot: destRoot, dryRun: dryRun, verifyMode: verifyMode)
            resultText = "任务已提交: \(accepted.jobID)\n状态: \(accepted.status)\n正在后台执行，界面会自动刷新进度。"
            currentJobID = accepted.jobID
            jobStatus = accepted.status
            totalCount = 0
            successCount = 0
            skippedCount = 0
            failedCount = 0
            consecutivePollFailures = 0
            etaText = "计算中"
            progressSamples.removeAll()
            startPolling(jobID: accepted.jobID)
        } catch {
            loading = false
            cancelling = false
            resultText = "导入失败: \(error.localizedDescription)"
        }
    }

    func stopCurrentTask() async {
        guard canStop else { return }
        cancelling = true
        do {
            try await APIClient.shared.cancelJob(jobID: currentJobID)
            resultText = "已请求中止任务，正在等待任务安全停止..."
        } catch {
            cancelling = false
            resultText = "中止失败: \(error.localizedDescription)"
        }
    }

    var canStop: Bool {
        !currentJobID.isEmpty && (jobStatus == "running" || jobStatus == "accepted") && !cancelling
    }

    private func startPolling(jobID: String) {
        pollingTask?.cancel()
        pollingTask = Task {
            defer { loading = false }
            var backoffNanos: UInt64 = 1_500_000_000
            let maxBackoff: UInt64 = 10_000_000_000
            let maxFailures = 8
            var startupGraceRetries = 0

            while !Task.isCancelled {
                do {
                    let detail = try await APIClient.shared.jobDetail(id: jobID)
                    let j = detail.job
                    consecutivePollFailures = 0
                    backoffNanos = 1_500_000_000

                    currentJobID = j.id
                    jobStatus = j.status
                    totalCount = j.totalCount
                    successCount = j.successCount
                    skippedCount = j.skippedCount
                    failedCount = j.failedCount
                    updateETA(status: j.status)

                    resultText = "job=\(j.id) status=\(j.status) total=\(j.totalCount) success=\(j.successCount) skipped=\(j.skippedCount) failed=\(j.failedCount)"
                    if j.status != "running" {
                        etaText = "已完成"
                        cancelling = false
                        return
                    }
                } catch {
                    if isStartupNotReady(error), startupGraceRetries < 6 {
                        startupGraceRetries += 1
                        resultText = "任务初始化中，正在准备导入..."
                        try? await Task.sleep(nanoseconds: 700_000_000)
                        continue
                    }

                    consecutivePollFailures += 1
                    if consecutivePollFailures > maxFailures {
                        resultText = "任务查询失败（已多次重试）: \(error.localizedDescription)"
                        cancelling = false
                        return
                    }
                    resultText = "任务查询失败，\(Double(backoffNanos) / 1_000_000_000)s 后重试: \(error.localizedDescription)"
                    try? await Task.sleep(nanoseconds: backoffNanos)
                    backoffNanos = min(backoffNanos * 2, maxBackoff)
                    continue
                }
                try? await Task.sleep(nanoseconds: 1_500_000_000)
            }
        }
    }

    private func isStartupNotReady(_ error: Error) -> Bool {
        let msg = error.localizedDescription.lowercased()
        return msg.contains("no rows") || msg.contains("not found") || msg.contains("404")
    }

    private func updateETA(status: String) {
        let done = successCount + skippedCount + failedCount
        let now = Date()
        progressSamples.append((time: now, done: done))
        if progressSamples.count > 12 {
            progressSamples.removeFirst(progressSamples.count - 12)
        }

        if status != "running" {
            etaText = "已完成"
            return
        }
        if totalCount <= 0 {
            etaText = "计算中"
            return
        }
        let remain = totalCount - done
        if remain <= 0 {
            etaText = "即将完成"
            return
        }
        guard let first = progressSamples.first, let last = progressSamples.last else {
            etaText = "计算中"
            return
        }
        let deltaDone = last.done - first.done
        let deltaTime = last.time.timeIntervalSince(first.time)
        if deltaDone <= 0 || deltaTime < 1 {
            etaText = "计算中"
            return
        }
        let speed = Double(deltaDone) / deltaTime
        if speed <= 0 {
            etaText = "计算中"
            return
        }
        etaText = formatETA(seconds: Double(remain) / speed)
    }

    private func formatETA(seconds: Double) -> String {
        if !seconds.isFinite || seconds <= 1 { return "< 1 秒" }
        let total = Int(seconds.rounded())
        let h = total / 3600
        let m = (total % 3600) / 60
        let s = total % 60
        if h > 0 { return "\(h)小时\(m)分" }
        if m > 0 { return "\(m)分\(s)秒" }
        return "\(s)秒"
    }
}
