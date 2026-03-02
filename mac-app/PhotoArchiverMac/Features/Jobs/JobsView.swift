import SwiftUI

struct JobsView: View {
    @State private var jobs: [JobSummary] = []
    @State private var selected: JobSummary?
    @State private var detailText: String = ""
    @State private var loading: Bool = false
    @State private var showDeleteConfirm: Bool = false
    private let pollTimer = Timer.publish(every: 2.0, on: .main, in: .common).autoconnect()

    var body: some View {
        HStack(spacing: 0) {
            VStack(alignment: .leading) {
                HStack {
                    Text("任务记录")
                        .font(.title2)
                        .bold()
                    Spacer()
                    Button("中止任务") {
                        guard let id = selected?.id else { return }
                        Task { await cancelJob(id: id) }
                    }
                    .disabled(selected?.status != "running" || loading)
                    Button("删除任务") {
                        showDeleteConfirm = true
                    }
                    .disabled(selected == nil || loading)
                    Button("刷新") { Task { await loadJobs() } }
                }
                List(jobs, selection: $selected) { job in
                    VStack(alignment: .leading) {
                        Text(job.id).font(.caption)
                        HStack(spacing: 8) {
                            StatusBadge(status: job.status)
                            Text("total \(job.totalCount)")
                                .foregroundStyle(.secondary)
                                .font(.caption)
                            Text("failed \(job.failedCount)")
                                .foregroundStyle(.secondary)
                                .font(.caption)
                        }
                    }
                    .tag(job)
                }
            }
            .frame(minWidth: 360)
            Divider()
            VStack(alignment: .leading, spacing: 12) {
                Text("任务详情")
                    .font(.title3)
                    .bold()
                if let selected {
                    Button("重试失败项") { Task { await retryFailed(jobID: selected.id) } }
                        .disabled(loading)
                    ScrollView {
                        Text(detailText)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                } else {
                    Text("请选择任务")
                }
                Spacer()
            }
            .padding()
        }
        .padding()
        .task { await loadJobs() }
        .alert("确认删除任务", isPresented: $showDeleteConfirm) {
            Button("删除", role: .destructive) {
                guard let id = selected?.id else { return }
                Task { await deleteJob(id: id) }
            }
            Button("取消", role: .cancel) {}
        } message: {
            Text("将删除任务记录和该任务条目，无法恢复。")
        }
        .onReceive(pollTimer) { _ in
            guard shouldPoll else { return }
            Task { await loadJobs() }
        }
        .onChange(of: selected?.id) { _ in
            if let id = selected?.id {
                Task { await loadJobDetail(id: id) }
            }
        }
    }

    private func loadJobs() async {
        loading = true
        defer { loading = false }
        do {
            jobs = try await APIClient.shared.listJobs(limit: 30)
        } catch {
            detailText = "加载任务失败: \(error.localizedDescription)"
        }
    }

    private func loadJobDetail(id: String) async {
        loading = true
        defer { loading = false }
        do {
            let detail = try await APIClient.shared.jobDetail(id: id)
            detailText = "job=\(detail.job.id)\nstatus=\(detail.job.status)\ntotal=\(detail.job.totalCount)\nsuccess=\(detail.job.successCount)\nskipped=\(detail.job.skippedCount)\nfailed=\(detail.job.failedCount)\n当前展示条目=\(detail.items.count)（为保护性能，详情列表有返回上限）"
        } catch {
            detailText = "加载详情失败: \(error.localizedDescription)"
        }
    }

    private func retryFailed(jobID: String) async {
        loading = true
        defer { loading = false }
        do {
            let res = try await APIClient.shared.retryFailed(jobID: jobID, dryRun: false, verifyMode: "size", async: false)
            if let r = res.result {
                detailText = "重试完成: job=\(r.jobID) success=\(r.successCount) failed=\(r.failedCount)"
            } else if let newID = res.jobID {
                detailText = "重试任务已创建: \(newID)"
            } else {
                detailText = "已发起重试"
            }
            await loadJobs()
        } catch {
            detailText = "重试失败: \(error.localizedDescription)"
        }
    }

    private func deleteJob(id: String) async {
        loading = true
        defer { loading = false }
        do {
            try await APIClient.shared.deleteJob(jobID: id)
            detailText = "任务已删除: \(id)"
            selected = nil
            await loadJobs()
        } catch {
            detailText = "删除失败: \(error.localizedDescription)"
        }
    }

    private func cancelJob(id: String) async {
        loading = true
        defer { loading = false }
        do {
            try await APIClient.shared.cancelJob(jobID: id)
            detailText = "已请求中止任务: \(id)"
            try? await Task.sleep(nanoseconds: 700_000_000)
            await loadJobs()
            if selected?.id == id {
                await loadJobDetail(id: id)
            }
        } catch {
            detailText = "中止失败: \(error.localizedDescription)"
        }
    }

    private var shouldPoll: Bool {
        jobs.contains { $0.status == "running" }
    }
}

private struct StatusBadge: View {
    let status: String

    var body: some View {
        Text(status)
            .font(.caption2)
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .background(color.opacity(0.15))
            .foregroundStyle(color)
            .clipShape(Capsule())
    }

    private var color: Color {
        switch status {
        case "success":
            return .green
        case "partial_failed":
            return .orange
        case "failed":
            return .red
        case "running":
            return .blue
        case "cancelled":
            return .orange
        default:
            return .gray
        }
    }
}

extension JobSummary: Hashable {
    static func == (lhs: JobSummary, rhs: JobSummary) -> Bool { lhs.id == rhs.id }
    func hash(into hasher: inout Hasher) { hasher.combine(id) }
}
