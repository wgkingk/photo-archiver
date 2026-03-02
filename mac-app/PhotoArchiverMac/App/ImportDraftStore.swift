import Foundation

@MainActor
final class ImportDraftStore: ObservableObject {
    @Published var sourceRoot: String = ""
    @Published var destRoot: String = ""
    @Published var verifyMode: String = "size"
    @Published var dryRun: Bool = true
}
