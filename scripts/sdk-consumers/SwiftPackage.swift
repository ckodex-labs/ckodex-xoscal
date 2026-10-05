// swift-tools-version: 6.0
import PackageDescription
let package = Package(name: "Smoke", dependencies: [.package(path: "/module")],
 targets: [.executableTarget(name: "Smoke", dependencies: [.product(name: "XoscalSDK", package: "module")])])
