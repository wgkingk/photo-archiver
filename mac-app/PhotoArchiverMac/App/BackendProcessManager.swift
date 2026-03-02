import Foundation

@MainActor
final class BackendProcessManager: ObservableObject {
    @Published private(set) var isReady: Bool = false
    @Published private(set) var statusText: String = "正在启动本地服务..."
    @Published private(set) var baseURL: URL = URL(string: "http://127.0.0.1:38080")!
    @Published private(set) var currentPort: Int?
    @Published private(set) var backendPID: Int32?

    private var process: Process?

    func start() async {
        if isReady { return }

        let ports = Array(38080...38090)
        for port in ports {
            do {
                try startProcess(port: port)
            } catch {
                statusText = "服务启动失败：\(error.localizedDescription)"
                continue
            }

            let url = URL(string: "http://127.0.0.1:\(port)")!
            APIClient.shared.configure(baseURL: url)

            let ok = await waitForHealthReady(retry: 25, intervalNanos: 200_000_000)
            if ok {
                baseURL = url
                isReady = true
                currentPort = port
                backendPID = process?.processIdentifier
                statusText = "服务已就绪（\(url.host ?? "127.0.0.1"):\(url.port ?? port)）"
                return
            }

            stop()
        }

        statusText = "无法启动本地服务，请检查端口占用或后端资源是否存在"
    }

    func stop() {
        guard let process else { return }
        if process.isRunning {
            process.terminate()
        }
        self.process = nil
        isReady = false
        currentPort = nil
        backendPID = nil
        statusText = "服务已停止"
    }

    func restart() async {
        stop()
        statusText = "正在重启本地服务..."
        await start()
    }

    private func startProcess(port: Int) throws {
        let executable = Bundle.main.url(forResource: "photo-archiver-service", withExtension: nil, subdirectory: "backend")
        let schema = Bundle.main.url(forResource: "DB_SCHEMA", withExtension: "sql", subdirectory: "backend")
        guard let executable, let schema else {
            throw NSError(domain: "BackendProcessManager", code: -11, userInfo: [NSLocalizedDescriptionKey: "缺少内置后端资源，请重新构建应用"])
        }

        let appSupport = try appSupportDirectory()
        try FileManager.default.createDirectory(at: appSupport, withIntermediateDirectories: true)
        let dbPath = appSupport.appendingPathComponent("photo_archiver.db").path

        let p = Process()
        p.executableURL = executable
        p.arguments = [
            "--addr", "127.0.0.1:\(port)",
            "--db", dbPath,
            "--schema", schema.path,
        ]
        p.standardOutput = Pipe()
        p.standardError = Pipe()
        try p.run()
        self.process = p
    }

    private func waitForHealthReady(retry: Int, intervalNanos: UInt64) async -> Bool {
        for _ in 0..<retry {
            if await APIClient.shared.health() {
                return true
            }
            try? await Task.sleep(nanoseconds: intervalNanos)
            if process?.isRunning == false {
                return false
            }
        }
        return false
    }

    private func appSupportDirectory() throws -> URL {
        let base = try FileManager.default.url(for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        return base.appendingPathComponent("PhotoArchiver", isDirectory: true)
    }
}
