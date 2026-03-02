// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "PhotoArchiverMac",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "PhotoArchiverMac", targets: ["PhotoArchiverMac"])
    ],
    targets: [
        .executableTarget(
            name: "PhotoArchiverMac",
            path: "PhotoArchiverMac"
        )
    ]
)
