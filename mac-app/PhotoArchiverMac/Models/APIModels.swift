import Foundation

struct ScanRequest: Codable {
    let sourceRoot: String

    enum CodingKeys: String, CodingKey {
        case sourceRoot = "source_root"
    }
}

struct ScanResponse: Codable {
    let totalCount: Int
    let totalBytes: Int64
    let exifCount: Int
    let fallbackCount: Int

    enum CodingKeys: String, CodingKey {
        case totalCount = "total_count"
        case totalBytes = "total_bytes"
        case exifCount = "exif_count"
        case fallbackCount = "fallback_count"
    }
}

struct ImportRequestDTO: Codable {
    let sourceRoot: String
    let destRoot: String
    let dryRun: Bool
    let verifyMode: String
    let async: Bool

    enum CodingKeys: String, CodingKey {
        case sourceRoot = "source_root"
        case destRoot = "dest_root"
        case dryRun = "dry_run"
        case verifyMode = "verify_mode"
        case async
    }
}

struct ImportResult: Codable {
    let jobID: String
    let totalCount: Int
    let successCount: Int
    let skippedCount: Int
    let failedCount: Int
    let totalBytes: Int64
    let copiedBytes: Int64
    let status: String

    enum CodingKeys: String, CodingKey {
        case jobID = "JobID"
        case totalCount = "TotalCount"
        case successCount = "SuccessCount"
        case skippedCount = "SkippedCount"
        case failedCount = "FailedCount"
        case totalBytes = "TotalBytes"
        case copiedBytes = "CopiedBytes"
        case status = "Status"
    }
}

struct ImportAcceptedResponse: Codable {
    let jobID: String
    let status: String

    enum CodingKeys: String, CodingKey {
        case jobID = "job_id"
        case status
    }
}

struct JobSummary: Codable, Identifiable {
    let id: String
    let status: String
    let totalCount: Int
    let successCount: Int
    let skippedCount: Int
    let failedCount: Int
    let totalBytes: Int64
    let copiedBytes: Int64
    let startedAt: String
    let finishedAt: String

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case status = "Status"
        case totalCount = "TotalCount"
        case successCount = "SuccessCount"
        case skippedCount = "SkippedCount"
        case failedCount = "FailedCount"
        case totalBytes = "TotalBytes"
        case copiedBytes = "CopiedBytes"
        case startedAt = "StartedAt"
        case finishedAt = "FinishedAt"
    }
}

struct JobsResponse: Codable {
    let items: [JobSummary]
}

struct JobItem: Codable, Identifiable {
    let id: String
    let sourcePath: String
    let targetPath: String
    let status: String
    let reason: String
    let sizeBytes: Int64

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case sourcePath = "SourcePath"
        case targetPath = "TargetPath"
        case status = "Status"
        case reason = "Reason"
        case sizeBytes = "SizeBytes"
    }
}

struct JobDetailResponse: Codable {
    let job: JobSummary
    let items: [JobItem]
}

struct RetryFailedRequest: Codable {
    let dryRun: Bool
    let verifyMode: String
    let async: Bool

    enum CodingKeys: String, CodingKey {
        case dryRun = "dry_run"
        case verifyMode = "verify_mode"
        case async
    }
}

struct RetryFailedResponse: Codable {
    let sourceJobID: String?
    let retryCount: Int?
    let result: ImportResult?
    let jobID: String?

    enum CodingKeys: String, CodingKey {
        case sourceJobID = "source_job_id"
        case retryCount = "retry_count"
        case result
        case jobID = "job_id"
    }
}
