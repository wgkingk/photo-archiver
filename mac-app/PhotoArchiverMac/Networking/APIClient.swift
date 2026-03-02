import Foundation

@MainActor
final class APIClient {
    static let shared = APIClient()
    private var baseURL = URL(string: "http://127.0.0.1:38080")!

    private init() {}

    func configure(baseURL: URL) {
        self.baseURL = baseURL
    }

    func health() async -> Bool {
        do {
            struct Health: Codable { let status: String }
            let res: Health = try await request(path: "/health", method: "GET", body: Optional<String>.none)
            return res.status == "ok"
        } catch {
            return false
        }
    }

    func scan(sourceRoot: String) async throws -> ScanResponse {
        try await request(
            path: "/v1/scan",
            method: "POST",
            body: ScanRequest(sourceRoot: sourceRoot)
        )
    }

    func startImport(sourceRoot: String, destRoot: String, dryRun: Bool, verifyMode: String, async: Bool) async throws -> ImportResult {
        try await request(
            path: "/v1/import",
            method: "POST",
            body: ImportRequestDTO(sourceRoot: sourceRoot, destRoot: destRoot, dryRun: dryRun, verifyMode: verifyMode, async: async)
        )
    }

    func startImportAsync(sourceRoot: String, destRoot: String, dryRun: Bool, verifyMode: String) async throws -> ImportAcceptedResponse {
        try await request(
            path: "/v1/import",
            method: "POST",
            body: ImportRequestDTO(sourceRoot: sourceRoot, destRoot: destRoot, dryRun: dryRun, verifyMode: verifyMode, async: true)
        )
    }

    func listJobs(limit: Int = 30) async throws -> [JobSummary] {
        let response: JobsResponse = try await request(path: "/v1/jobs", query: [URLQueryItem(name: "limit", value: "\(limit)")], method: "GET", body: Optional<String>.none)
        return response.items
    }

    func jobDetail(id: String) async throws -> JobDetailResponse {
        try await request(path: "/v1/jobs/\(id)", method: "GET", body: Optional<String>.none)
    }

    func retryFailed(jobID: String, dryRun: Bool, verifyMode: String, async: Bool) async throws -> RetryFailedResponse {
        try await request(
            path: "/v1/jobs/\(jobID)/retry-failed",
            method: "POST",
            body: RetryFailedRequest(dryRun: dryRun, verifyMode: verifyMode, async: async)
        )
    }

    func deleteJob(jobID: String) async throws {
        struct DeleteResponse: Codable { let status: String }
        let _: DeleteResponse = try await request(
            path: "/v1/jobs/\(jobID)",
            method: "DELETE",
            body: Optional<String>.none
        )
    }

    func cancelJob(jobID: String) async throws {
        struct CancelResponse: Codable { let status: String }
        let _: CancelResponse = try await request(
            path: "/v1/jobs/\(jobID)/cancel",
            method: "POST",
            body: Optional<String>.none
        )
    }

    private func request<T: Decodable, U: Encodable>(path: String, query: [URLQueryItem] = [], method: String, body: U?) async throws -> T {
        guard var comp = URLComponents(url: baseURL.appendingPathComponent(path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))), resolvingAgainstBaseURL: false) else {
            throw NSError(domain: "APIClient", code: -2, userInfo: [NSLocalizedDescriptionKey: "Invalid URL"])
        }
        comp.queryItems = query.isEmpty ? nil : query
        guard let url = comp.url else {
            throw NSError(domain: "APIClient", code: -3, userInfo: [NSLocalizedDescriptionKey: "Invalid URL components"])
        }
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        if let body {
            req.httpBody = try JSONEncoder().encode(body)
        }
        let (data, resp) = try await URLSession.shared.data(for: req)
        guard let httpResp = resp as? HTTPURLResponse else {
            throw NSError(domain: "APIClient", code: -1, userInfo: [NSLocalizedDescriptionKey: "Invalid response"])
        }
        guard (200...299).contains(httpResp.statusCode) else {
            let text = String(data: data, encoding: .utf8) ?? "Request failed"
            throw NSError(domain: "APIClient", code: httpResp.statusCode, userInfo: [NSLocalizedDescriptionKey: text])
        }
        return try JSONDecoder().decode(T.self, from: data)
    }
}
